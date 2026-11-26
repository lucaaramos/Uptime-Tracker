package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lv99/uptime-tracker/internal/config"
	"github.com/lv99/uptime-tracker/internal/messaging"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IncidentStore interface {
	OpenOrUpdate(ctx context.Context, monitorID primitive.ObjectID, monitorName, url, cause string, statusCode, failCount int, at time.Time) (*model.Incident, bool, error)
	ResolveOpen(ctx context.Context, monitorID primitive.ObjectID, at time.Time) (*model.Incident, error)
}

type Worker struct {
	cfg       *config.Config
	check     *repository.CheckRepo
	monitor   *repository.MonitorRepo
	incidents IncidentStore
	mq        *messaging.RabbitMQ
	rdb       *redis.Client
	client    *http.Client
}

func New(cfg *config.Config, check *repository.CheckRepo, monitor *repository.MonitorRepo, incidents IncidentStore, mq *messaging.RabbitMQ, rdb *redis.Client) *Worker {
	return &Worker{
		cfg:       cfg,
		check:     check,
		monitor:   monitor,
		incidents: incidents,
		mq:        mq,
		rdb:       rdb,
		client:    newHTTPClient(cfg.AllowPrivateURLs),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	msgs, err := w.mq.Consume(messaging.QueuePending)
	if err != nil {
		return fmt.Errorf("worker: failed to consume queue: %w", err)
	}

	log.Info().Int("count", w.cfg.WorkerCount).Msg("worker pool started")

	sem := make(chan struct{}, w.cfg.WorkerCount)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("worker stopped")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("worker: deliveries channel closed")
			}

			sem <- struct{}{}
			go func(d amqp.Delivery) {
				defer func() { <-sem }()
				w.processMessage(ctx, d)
			}(msg)
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, d amqp.Delivery) {
	var job model.CheckJob
	if err := json.Unmarshal(d.Body, &job); err != nil {
		log.Error().Err(err).Msg("worker: failed to unmarshal job")
		d.Nack(false, false)
		return
	}

	result := w.executeCheck(job)
	w.saveAndPublish(ctx, job, result)

	d.Ack(false)
}

func (w *Worker) executeCheck(job model.CheckJob) *model.CheckResult {
	result := &model.CheckResult{
		CheckedAt: time.Now(),
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(job.TimeoutMs)*time.Millisecond)
	defer cancel()

	method := strings.ToUpper(job.Method)
	if method == "" {
		method = "GET"
	}

	if err := validateTargetURL(job.URL); err != nil {
		result.Success = false
		result.ErrorMsg = err.Error()
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}

	req, err := http.NewRequestWithContext(ctx, method, job.URL, nil)
	if err != nil {
		result.Success = false
		result.ErrorMsg = fmt.Sprintf("request creation failed: %v", err)
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}

	req.Header.Set("User-Agent", "UptimeTracker/1.0")
	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		result.Success = false
		result.ErrorMsg = fmt.Sprintf("request failed: %v", err)
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)

	result.StatusCode = resp.StatusCode
	result.LatencyMs = time.Since(start).Milliseconds()
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	return result
}

func (w *Worker) saveAndPublish(ctx context.Context, job model.CheckJob, result *model.CheckResult) {
	monitorID, err := primitive.ObjectIDFromHex(job.MonitorID)
	if err != nil {
		log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: invalid monitor id")
		return
	}
	result.MonitorID = monitorID

	saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.check.Insert(saveCtx, result); err != nil {
		log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: failed to save check result")
	}

	// Cache last result
	cacheKey := fmt.Sprintf("monitor:%s:last_result", job.MonitorID)
	msgData, _ := json.Marshal(result)
	w.rdb.Set(saveCtx, cacheKey, msgData, 3600*time.Second)

	// Publish result message
	resultMsg := model.CheckResultMessage{
		MonitorID:  job.MonitorID,
		Success:    result.Success,
		StatusCode: result.StatusCode,
		LatencyMs:  result.LatencyMs,
		ErrorMsg:   result.ErrorMsg,
		CheckedAt:  result.CheckedAt.UnixMilli(),
	}

	resultData, err := json.Marshal(resultMsg)
	if err != nil {
		log.Error().Err(err).Msg("worker: failed to marshal result message")
		return
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.mq.Publish(pubCtx, messaging.RoutingKeyResult, resultData); err != nil {
		log.Error().Err(err).Msg("worker: failed to publish result")
	}

	if !result.Success {
		w.handleFailure(pubCtx, job, result)
	} else {
		w.handleRecovery(pubCtx, job, result)
	}
}

func failureCause(result *model.CheckResult) string {
	if result.ErrorMsg != "" {
		return result.ErrorMsg
	}
	if result.StatusCode > 0 {
		return fmt.Sprintf("HTTP %d", result.StatusCode)
	}
	return "check failed"
}

func shouldSendDownAlert(count int64) bool {
	return count == 1 || count == 3 || (count > 3 && ((count+1)&count) == 0)
}

func (w *Worker) handleFailure(ctx context.Context, job model.CheckJob, result *model.CheckResult) {
	failKey := fmt.Sprintf("monitor:%s:fail_count", job.MonitorID)

	count, err := w.rdb.Incr(ctx, failKey).Result()
	if err != nil {
		log.Error().Err(err).Msg("worker: failed to increment fail count")
		return
	}

	if count == 1 {
		w.rdb.Expire(ctx, failKey, 24*time.Hour)
	}

	monitorID, _ := primitive.ObjectIDFromHex(job.MonitorID)
	cause := failureCause(result)

	var monitor *model.Monitor
	if w.monitor != nil {
		m, err := w.monitor.GetByID(ctx, monitorID)
		if err != nil {
			log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: failed to get monitor for incident/alert")
		} else {
			monitor = m
		}
	}

	monitorName := job.URL
	if monitor != nil && monitor.Name != "" {
		monitorName = monitor.Name
	}

	var incidentID string
	if w.incidents != nil {
		inc, _, err := w.incidents.OpenOrUpdate(ctx, monitorID, monitorName, job.URL, cause, result.StatusCode, int(count), result.CheckedAt)
		if err != nil {
			log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: failed to record incident")
		} else if inc != nil {
			incidentID = inc.ID.Hex()
		}
	}

	if shouldSendDownAlert(count) {
		alertMsg := model.AlertMessage{
			Type:        model.AlertTypeDown,
			IncidentID:  incidentID,
			MonitorID:   job.MonitorID,
			MonitorName: monitorName,
			URL:         job.URL,
			FailCount:   int(count),
			LastError:   cause,
			Timestamp:   time.Now().UnixMilli(),
		}

		alertData, err := json.Marshal(alertMsg)
		if err != nil {
			log.Error().Err(err).Msg("worker: failed to marshal alert message")
			return
		}

		if err := w.mq.Publish(ctx, messaging.RoutingKeyAlert, alertData); err != nil {
			log.Error().Err(err).Msg("worker: failed to publish alert")
		}
	}
}

func (w *Worker) handleRecovery(ctx context.Context, job model.CheckJob, result *model.CheckResult) {
	failKey := fmt.Sprintf("monitor:%s:fail_count", job.MonitorID)

	prevCount, redisErr := w.rdb.Get(ctx, failKey).Int()
	if redisErr != nil && redisErr != redis.Nil {
		log.Error().Err(redisErr).Msg("worker: failed to read fail count")
	}
	w.rdb.Set(ctx, failKey, 0, 0)

	// Fast path: steady-state healthy monitor (fail_count was explicitly 0)
	if redisErr == nil && prevCount == 0 {
		return
	}

	monitorID, err := primitive.ObjectIDFromHex(job.MonitorID)
	if err != nil {
		return
	}

	var resolved *model.Incident
	if w.incidents != nil {
		resolved, err = w.incidents.ResolveOpen(ctx, monitorID, result.CheckedAt)
		if err != nil {
			log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: failed to resolve incident")
		}
	}

	if resolved == nil && prevCount == 0 {
		return
	}

	monitorName := job.URL
	var incidentID string
	var durationSec int64

	if resolved != nil {
		incidentID = resolved.ID.Hex()
		durationSec = resolved.DurationSec
		if resolved.MonitorName != "" {
			monitorName = resolved.MonitorName
		}
	} else if w.monitor != nil {
		if m, err := w.monitor.GetByID(ctx, monitorID); err == nil && m.Name != "" {
			monitorName = m.Name
		}
	}

	alertMsg := model.AlertMessage{
		Type:        model.AlertTypeRecovery,
		IncidentID:  incidentID,
		MonitorID:   job.MonitorID,
		MonitorName: monitorName,
		URL:         job.URL,
		FailCount:   prevCount,
		DurationSec: durationSec,
		Timestamp:   result.CheckedAt.UnixMilli(),
	}

	alertData, err := json.Marshal(alertMsg)
	if err != nil {
		log.Error().Err(err).Msg("worker: failed to marshal recovery alert")
		return
	}

	if err := w.mq.Publish(ctx, messaging.RoutingKeyAlert, alertData); err != nil {
		log.Error().Err(err).Msg("worker: failed to publish recovery alert")
	}
}

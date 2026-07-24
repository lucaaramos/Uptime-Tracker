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

type Worker struct {
	cfg     *config.Config
	check   *repository.CheckRepo
	monitor *repository.MonitorRepo
	mq      *messaging.RabbitMQ
	rdb     *redis.Client
	client  *http.Client
}

func New(cfg *config.Config, check *repository.CheckRepo, monitor *repository.MonitorRepo, mq *messaging.RabbitMQ, rdb *redis.Client) *Worker {
	return &Worker{
		cfg:     cfg,
		check:   check,
		monitor: monitor,
		mq:      mq,
		rdb:     rdb,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
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
		failKey := fmt.Sprintf("monitor:%s:fail_count", job.MonitorID)
		w.rdb.Set(pubCtx, failKey, 0, 0)
	}
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

	if count == 1 || count == 3 || (count > 3 && ((count+1)&count) == 0) {
		monitorID, _ := primitive.ObjectIDFromHex(job.MonitorID)
		monitor, err := w.monitor.GetByID(ctx, monitorID)
		if err != nil {
			log.Error().Err(err).Str("monitor_id", job.MonitorID).Msg("worker: failed to get monitor for alert")
			return
		}

		alertMsg := model.AlertMessage{
			MonitorID:   job.MonitorID,
			MonitorName: monitor.Name,
			URL:         job.URL,
			FailCount:   int(count),
			LastError:   result.ErrorMsg,
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

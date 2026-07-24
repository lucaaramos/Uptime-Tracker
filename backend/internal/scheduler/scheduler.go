package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lv99/uptime-tracker/internal/config"
	"github.com/lv99/uptime-tracker/internal/messaging"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	cfg     *config.Config
	monitor *repository.MonitorRepo
	check   *repository.CheckRepo
	mq      *messaging.RabbitMQ
	rdb     *redis.Client
}

func New(cfg *config.Config, monitor *repository.MonitorRepo, check *repository.CheckRepo, mq *messaging.RabbitMQ, rdb *redis.Client) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		monitor: monitor,
		check:   check,
		mq:      mq,
		rdb:     rdb,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	log.Info().Dur("interval", s.cfg.SchedulerInterval).Msg("scheduler started")

	ticker := time.NewTicker(s.cfg.SchedulerInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.tick(ctx)

	for {
		select {
		case <-ticker.C:
			s.tick(ctx)
		case <-ctx.Done():
			log.Info().Msg("scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	monitors, err := s.monitor.ListActive(ctx)
	if err != nil {
		log.Error().Err(err).Msg("scheduler: failed to list active monitors")
		return
	}

	for _, mon := range monitors {
		lastResult, err := s.check.GetLatestByMonitorID(ctx, mon.ID)
		if err != nil {
			// No previous result, should check now
			s.enqueueCheck(mon)
			continue
		}

		elapsed := time.Since(lastResult.CheckedAt)
		if elapsed >= time.Duration(mon.Interval)*time.Second {
			s.enqueueCheck(mon)
		}
	}
}

func (s *Scheduler) enqueueCheck(mon model.Monitor) {
	job := model.CheckJob{
		MonitorID: mon.ID.Hex(),
		URL:       mon.URL,
		Method:    mon.Method,
		Headers:   mon.Headers,
		TimeoutMs: mon.Timeout,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(job)
	if err != nil {
		log.Error().Err(err).Str("monitor_id", mon.ID.Hex()).Msg("scheduler: failed to marshal check job")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.mq.Publish(ctx, messaging.RoutingKeyPending, data); err != nil {
		log.Error().Err(err).Str("monitor_id", mon.ID.Hex()).Msg("scheduler: failed to publish check job")
		return
	}

	log.Debug().Str("monitor_id", mon.ID.Hex()).Str("url", mon.URL).Msg("scheduler: enqueued check")
}

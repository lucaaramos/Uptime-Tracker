package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lv99/uptime-tracker/internal/alerting"
	"github.com/lv99/uptime-tracker/internal/api"
	"github.com/lv99/uptime-tracker/internal/config"
	"github.com/lv99/uptime-tracker/internal/messaging"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/lv99/uptime-tracker/internal/scheduler"
	"github.com/lv99/uptime-tracker/internal/worker"
	"github.com/lv99/uptime-tracker/internal/ws"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mode := flag.String("mode", "all", "Run mode: api, worker, scheduler, all")
	flag.Parse()

	cfg := config.Load()

	// Logger
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		Level(parseLogLevel(cfg.LogLevel)).
		With().Timestamp().Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to mongodb")
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pingCancel()
	if err := mongoClient.Ping(pingCtx, nil); err != nil {
		log.Fatal().Err(err).Msg("failed to ping mongodb")
	}
	log.Info().Msg("mongodb connected")

	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database(cfg.MongoDB)
	monitorRepo := repository.NewMonitorRepo(db)
	checkRepo := repository.NewCheckRepo(db)
	authRepo := repository.NewAuthRepo(db)

	idxCtx, idxCancel := context.WithTimeout(ctx, 15*time.Second)
	defer idxCancel()
	if err := monitorRepo.EnsureIndexes(idxCtx); err != nil {
		log.Warn().Err(err).Msg("failed to ensure monitor indexes")
	}
	if err := checkRepo.EnsureIndexes(idxCtx, cfg.CheckRetentionDays); err != nil {
		log.Warn().Err(err).Msg("failed to ensure check indexes")
	}
	if err := authRepo.EnsureIndexes(idxCtx); err != nil {
		log.Warn().Err(err).Msg("failed to ensure auth indexes")
	}
	if err := authRepo.EnsureBootstrapAdmin(idxCtx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Warn().Err(err).Msg("failed to ensure bootstrap admin user")
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	pingCtxR, pingCancelR := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancelR()
	if err := rdb.Ping(pingCtxR).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	log.Info().Msg("redis connected")
	defer rdb.Close()

	// RabbitMQ
	mq, err := messaging.NewRabbitMQ(cfg.RabbitMQURI)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}
	defer mq.Close()

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Result consumer for WebSocket broadcast
	go consumeResults(ctx, mq, hub)

	switch *mode {
	case "api":
		runAPI(ctx, cfg, hub, monitorRepo, checkRepo, authRepo, rdb)
	case "worker":
		runWorker(ctx, cfg, monitorRepo, checkRepo, mq, rdb)
	case "scheduler":
		runScheduler(ctx, cfg, monitorRepo, checkRepo, mq, rdb)
	case "all":
		go runAPI(ctx, cfg, hub, monitorRepo, checkRepo, authRepo, rdb)
		go runWorker(ctx, cfg, monitorRepo, checkRepo, mq, rdb)
		go runScheduler(ctx, cfg, monitorRepo, checkRepo, mq, rdb)
		go runAlertWorker(ctx, cfg, monitorRepo, mq)
	default:
		log.Fatal().Str("mode", *mode).Msg("unknown mode")
	}

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info().Msg("shutting down...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info().Msg("shutdown complete")
}

func runAPI(ctx context.Context, cfg *config.Config, hub *ws.Hub, monitor *repository.MonitorRepo, check *repository.CheckRepo, auth *repository.AuthRepo, rdb *redis.Client) {
	srv := api.NewServer(cfg, hub, monitor, check, auth, rdb)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Handler(),
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("api server listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("api server error")
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)
}

func runWorker(ctx context.Context, cfg *config.Config, monitor *repository.MonitorRepo, check *repository.CheckRepo, mq *messaging.RabbitMQ, rdb *redis.Client) {
	w := worker.New(cfg, check, monitor, mq, rdb)
	if err := w.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("worker error")
	}
}

func runScheduler(ctx context.Context, cfg *config.Config, monitor *repository.MonitorRepo, check *repository.CheckRepo, mq *messaging.RabbitMQ, rdb *redis.Client) {
	s := scheduler.New(cfg, monitor, check, mq, rdb)
	s.Start(ctx)
}

func runAlertWorker(ctx context.Context, cfg *config.Config, monitor *repository.MonitorRepo, mq *messaging.RabbitMQ) {
	a := alerting.NewAlertWorker(
		mq, monitor,
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
		os.Getenv("SMTP_FROM"),
	)
	if err := a.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("alert worker error")
	}
}

func consumeResults(ctx context.Context, mq *messaging.RabbitMQ, hub *ws.Hub) {
	msgs, err := mq.Consume(messaging.QueueResults)
	if err != nil {
		log.Error().Err(err).Msg("failed to consume results for ws broadcast")
		return
	}

	log.Info().Msg("ws result consumer started")

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			hub.Broadcast(json.RawMessage(msg.Body))
			msg.Ack(false)
		}
	}
}

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

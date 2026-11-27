package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear environment
	os.Clearenv()

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("expected default mongo uri, got %s", cfg.MongoURI)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected default redis addr, got %s", cfg.RedisAddr)
	}
	if cfg.RabbitMQURI != "amqp://guest:guest@localhost:5672/" {
		t.Errorf("expected default rabbitmq uri, got %s", cfg.RabbitMQURI)
	}
	if cfg.JWTSecret != "dev-secret-change-in-production" {
		t.Errorf("expected default jwt secret, got %s", cfg.JWTSecret)
	}
	if cfg.SchedulerInterval != 10*time.Second {
		t.Errorf("expected 10s scheduler interval, got %v", cfg.SchedulerInterval)
	}
	if cfg.WorkerCount != 5 {
		t.Errorf("expected 5 workers, got %d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug log level, got %s", cfg.LogLevel)
	}
	if cfg.CheckRetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", cfg.CheckRetentionDays)
	}
	if cfg.AllowPrivateURLs {
		t.Errorf("expected AllowPrivateURLs false by default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "9090")
	os.Setenv("MONGO_URI", "mongodb://custom:27017")
	os.Setenv("REDIS_ADDR", "redis:6380")
	os.Setenv("RABBITMQ_URI", "amqp://admin:admin@rabbit:5672/")
	os.Setenv("JWT_SECRET", "my-custom-secret")
	os.Setenv("SCHEDULER_INTERVAL", "30s")
	os.Setenv("WORKER_COUNT", "10")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("CHECK_RETENTION_DAYS", "90")
	os.Setenv("ALLOW_PRIVATE_URLS", "true")
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "secret123")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://custom:27017" {
		t.Errorf("expected custom mongo uri, got %s", cfg.MongoURI)
	}
	if cfg.RedisAddr != "redis:6380" {
		t.Errorf("expected custom redis addr, got %s", cfg.RedisAddr)
	}
	if cfg.RabbitMQURI != "amqp://admin:admin@rabbit:5672/" {
		t.Errorf("expected custom rabbitmq uri, got %s", cfg.RabbitMQURI)
	}
	if cfg.JWTSecret != "my-custom-secret" {
		t.Errorf("expected custom jwt secret, got %s", cfg.JWTSecret)
	}
	if cfg.SchedulerInterval != 30*time.Second {
		t.Errorf("expected 30s scheduler interval, got %v", cfg.SchedulerInterval)
	}
	if cfg.WorkerCount != 10 {
		t.Errorf("expected 10 workers, got %d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected info log level, got %s", cfg.LogLevel)
	}
	if cfg.CheckRetentionDays != 90 {
		t.Errorf("expected 90 retention days, got %d", cfg.CheckRetentionDays)
	}
	if !cfg.AllowPrivateURLs {
		t.Errorf("expected AllowPrivateURLs true")
	}
	if cfg.AdminUsername != "admin" || cfg.AdminPassword != "secret123" {
		t.Errorf("expected admin credentials to load from env")
	}
}

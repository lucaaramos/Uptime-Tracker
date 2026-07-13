package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	RedisAddr      string
	RedisPassword  string
	RabbitMQURI    string
	JWTSecret      string
	SchedulerInterval time.Duration
	WorkerCount    int
	LogLevel       string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		Port:              getEnv("PORT", "8080"),
		MongoURI:          getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:           getEnv("MONGO_DB", "uptime_tracker"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RabbitMQURI:       getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		SchedulerInterval: getDuration("SCHEDULER_INTERVAL", 10*time.Second),
		WorkerCount:       getInt("WORKER_COUNT", 5),
		LogLevel:          getEnv("LOG_LEVEL", "debug"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return fallback
}

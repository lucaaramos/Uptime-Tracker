package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CheckResult struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID  primitive.ObjectID `bson:"monitor_id" json:"monitor_id"`
	StatusCode int                `bson:"status_code" json:"status_code"`
	LatencyMs  int64              `bson:"latency_ms" json:"latency_ms"`
	Success    bool               `bson:"success" json:"success"`
	ErrorMsg   string             `bson:"error_msg,omitempty" json:"error_msg,omitempty"`
	CheckedAt  time.Time          `bson:"checked_at" json:"checked_at"`
}

type CheckJob struct {
	MonitorID string `json:"monitor_id"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	TimeoutMs int    `json:"timeout_ms"`
	Timestamp int64  `json:"timestamp"`
}

type CheckResultMessage struct {
	MonitorID  string `json:"monitor_id"`
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	ErrorMsg   string `json:"error_msg,omitempty"`
	CheckedAt  int64  `json:"checked_at"`
}

type AlertMessage struct {
	MonitorID   string `json:"monitor_id"`
	MonitorName string `json:"monitor_name"`
	URL         string `json:"url"`
	FailCount   int    `json:"fail_count"`
	LastError   string `json:"last_error"`
	Timestamp   int64  `json:"timestamp"`
}

type Stats struct {
	UptimePercent  float64 `json:"uptime_percent"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	TotalChecks    int     `json:"total_checks"`
	SuccessChecks  int     `json:"success_checks"`
	FailureChecks  int     `json:"failure_checks"`
}

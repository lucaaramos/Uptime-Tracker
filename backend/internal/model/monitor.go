package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Monitor struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	URL          string             `bson:"url" json:"url"`
	Method       string             `bson:"method" json:"method"`
	Interval     int                `bson:"interval_s" json:"interval_s"`
	Timeout      int                `bson:"timeout_ms" json:"timeout_ms"`
	Status       string             `bson:"status" json:"status"`
	AlertEmail   string             `bson:"alert_email,omitempty" json:"alert_email,omitempty"`
	AlertWebhook string             `bson:"alert_webhook,omitempty" json:"alert_webhook,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

type MonitorRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	Interval     int    `json:"interval_s"`
	Timeout      int    `json:"timeout_ms"`
	AlertEmail   string `json:"alert_email,omitempty"`
	AlertWebhook string `json:"alert_webhook,omitempty"`
}

type MonitorResponse struct {
	Monitor    Monitor `json:"monitor"`
	LastResult *CheckResult `json:"last_result,omitempty"`
	Uptime24h  float64 `json:"uptime_24h,omitempty"`
}

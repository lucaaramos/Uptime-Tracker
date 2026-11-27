package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	IncidentStatusOpen     = "open"
	IncidentStatusResolved = "resolved"

	AlertTypeDown     = "down"
	AlertTypeRecovery = "recovery"
)

type Incident struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MonitorID   primitive.ObjectID `bson:"monitor_id" json:"monitor_id"`
	MonitorName string             `bson:"monitor_name" json:"monitor_name"`
	URL         string             `bson:"url" json:"url"`
	Status      string             `bson:"status" json:"status"`
	Cause       string             `bson:"cause" json:"cause"`
	StatusCode  int                `bson:"status_code,omitempty" json:"status_code,omitempty"`
	FailCount   int                `bson:"fail_count" json:"fail_count"`
	StartedAt   time.Time          `bson:"started_at" json:"started_at"`
	ResolvedAt  *time.Time         `bson:"resolved_at,omitempty" json:"resolved_at,omitempty"`
	DurationSec int64              `bson:"duration_s,omitempty" json:"duration_s,omitempty"`
}

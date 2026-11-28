package model

import "time"

const (
	OverallStatusOperational = "operational"
	OverallStatusDegraded    = "degraded"
	OverallStatusMajorOutage = "major_outage"
	OverallStatusNoMonitors  = "no_monitors"
)

type DailyUptime struct {
	Date          string  `json:"date"`
	TotalChecks   int     `json:"total_checks"`
	SuccessChecks int     `json:"success_checks"`
	UptimePercent float64 `json:"uptime_percent"`
}

type PublicMonitorStatus struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Status        string        `json:"status"` // "up" | "down" | "pending"
	UptimePercent float64       `json:"uptime_percent"`
	AvgLatencyMs  float64       `json:"avg_latency_ms"`
	LastCheckedAt *time.Time    `json:"last_checked_at,omitempty"`
	Daily         []DailyUptime `json:"daily"`
}

type PublicIncident struct {
	ID          string     `json:"id"`
	MonitorID   string     `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	Status      string     `json:"status"`
	Cause       string     `json:"cause"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	DurationSec int64      `json:"duration_s,omitempty"`
}

type PublicStatusResponse struct {
	OverallStatus   string                `json:"overall_status"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Days            int                   `json:"days"`
	Monitors        []PublicMonitorStatus `json:"monitors"`
	RecentIncidents []PublicIncident      `json:"recent_incidents"`
}

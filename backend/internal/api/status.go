package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PublicMonitorStore interface {
	ListPublic(ctx context.Context) ([]model.Monitor, error)
}

type StatusCheckStore interface {
	GetLatestByMonitorID(ctx context.Context, monitorID primitive.ObjectID) (*model.CheckResult, error)
	GetStats(ctx context.Context, monitorID primitive.ObjectID, since time.Time) (*model.Stats, error)
	GetDailyUptime(ctx context.Context, monitorID primitive.ObjectID, days int) ([]model.DailyUptime, error)
}

type StatusHandler struct {
	monitors  PublicMonitorStore
	checks    StatusCheckStore
	incidents IncidentLister
}

func NewStatusHandler(monitors PublicMonitorStore, checks StatusCheckStore, incidents IncidentLister) *StatusHandler {
	return &StatusHandler{
		monitors:  monitors,
		checks:    checks,
		incidents: incidents,
	}
}

func (h *StatusHandler) PublicStatus(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil {
		if d >= 7 && d <= 90 {
			days = d
		}
	}

	monitors, err := h.monitors.ListPublic(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list public monitors")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load status"})
		return
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	publicIDs := make(map[primitive.ObjectID]bool, len(monitors))
	items := make([]model.PublicMonitorStatus, 0, len(monitors))

	downCount := 0
	for _, m := range monitors {
		publicIDs[m.ID] = true

		item := model.PublicMonitorStatus{
			ID:     m.ID.Hex(),
			Name:   m.Name,
			Status: "pending",
		}

		if last, err := h.checks.GetLatestByMonitorID(r.Context(), m.ID); err == nil && last != nil {
			checkedAt := last.CheckedAt
			item.LastCheckedAt = &checkedAt
			if last.Success {
				item.Status = "up"
			} else {
				item.Status = "down"
				downCount++
			}
		}

		if stats, err := h.checks.GetStats(r.Context(), m.ID, since); err == nil && stats != nil {
			item.UptimePercent = stats.UptimePercent
			item.AvgLatencyMs = stats.AvgLatencyMs
		}

		if daily, err := h.checks.GetDailyUptime(r.Context(), m.ID, days); err == nil {
			item.Daily = daily
		} else {
			item.Daily = []model.DailyUptime{}
		}

		items = append(items, item)
	}

	overall := model.OverallStatusOperational
	if len(items) == 0 {
		overall = model.OverallStatusNoMonitors
	} else if downCount == len(items) {
		overall = model.OverallStatusMajorOutage
	} else if downCount > 0 {
		overall = model.OverallStatusDegraded
	}

	recentIncidents := make([]model.PublicIncident, 0)
	if h.incidents != nil && len(publicIDs) > 0 {
		if allIncidents, err := h.incidents.ListRecent(r.Context(), "", 50, 0); err == nil {
			for _, inc := range allIncidents {
				if !publicIDs[inc.MonitorID] {
					continue
				}
				recentIncidents = append(recentIncidents, model.PublicIncident{
					ID:          inc.ID.Hex(),
					MonitorID:   inc.MonitorID.Hex(),
					MonitorName: inc.MonitorName,
					Status:      inc.Status,
					Cause:       inc.Cause,
					StartedAt:   inc.StartedAt,
					ResolvedAt:  inc.ResolvedAt,
					DurationSec: inc.DurationSec,
				})
				if len(recentIncidents) >= 10 {
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, model.PublicStatusResponse{
		OverallStatus:   overall,
		UpdatedAt:       time.Now().UTC(),
		Days:            days,
		Monitors:        items,
		RecentIncidents: recentIncidents,
	})
}

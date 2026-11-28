package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubPublicMonitorStore struct {
	monitors []model.Monitor
}

func (s *stubPublicMonitorStore) ListPublic(_ context.Context) ([]model.Monitor, error) {
	return s.monitors, nil
}

type stubStatusCheckStore struct {
	latest map[primitive.ObjectID]*model.CheckResult
	stats  map[primitive.ObjectID]*model.Stats
	daily  map[primitive.ObjectID][]model.DailyUptime
}

func (s *stubStatusCheckStore) GetLatestByMonitorID(_ context.Context, id primitive.ObjectID) (*model.CheckResult, error) {
	if r, ok := s.latest[id]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

func (s *stubStatusCheckStore) GetStats(_ context.Context, id primitive.ObjectID, _ time.Time) (*model.Stats, error) {
	if st, ok := s.stats[id]; ok {
		return st, nil
	}
	return &model.Stats{}, nil
}

func (s *stubStatusCheckStore) GetDailyUptime(_ context.Context, id primitive.ObjectID, days int) ([]model.DailyUptime, error) {
	if d, ok := s.daily[id]; ok {
		return d, nil
	}
	return make([]model.DailyUptime, days), nil
}

func TestPublicStatusOverallStatesAndFiltering(t *testing.T) {
	pub1 := primitive.NewObjectID()
	pub2 := primitive.NewObjectID()
	priv := primitive.NewObjectID()
	now := time.Now().UTC()

	incidents := &stubIncidentLister{
		incidents: []model.Incident{
			{
				ID:          primitive.NewObjectID(),
				MonitorID:   pub1,
				MonitorName: "Public API",
				Status:      model.IncidentStatusResolved,
				Cause:       "HTTP 503",
				StartedAt:   now.Add(-time.Hour),
				DurationSec: 60,
			},
			{
				ID:          primitive.NewObjectID(),
				MonitorID:   priv,
				MonitorName: "Private Internal DB",
				Status:      model.IncidentStatusOpen,
				Cause:       "connection refused",
				StartedAt:   now,
			},
		},
	}

	// 1. Degraded: 1 up, 1 down; private incident must be filtered out
	h := NewStatusHandler(
		&stubPublicMonitorStore{
			monitors: []model.Monitor{
				{ID: pub1, Name: "Public API", Public: true, Status: "active"},
				{ID: pub2, Name: "Checkout", Public: true, Status: "active"},
			},
		},
		&stubStatusCheckStore{
			latest: map[primitive.ObjectID]*model.CheckResult{
				pub1: {Success: true, CheckedAt: now},
				pub2: {Success: false, CheckedAt: now},
			},
			stats: map[primitive.ObjectID]*model.Stats{
				pub1: {UptimePercent: 99.9, AvgLatencyMs: 120},
				pub2: {UptimePercent: 85.0, AvgLatencyMs: 450},
			},
		},
		incidents,
	)

	req := httptest.NewRequest("GET", "/api/v1/status?days=30", nil)
	w := httptest.NewRecorder()
	h.PublicStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.PublicStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.OverallStatus != model.OverallStatusDegraded {
		t.Errorf("expected degraded, got %s", resp.OverallStatus)
	}
	if len(resp.Monitors) != 2 {
		t.Fatalf("expected 2 monitors, got %d", len(resp.Monitors))
	}
	if len(resp.RecentIncidents) != 1 || resp.RecentIncidents[0].MonitorName != "Public API" {
		t.Errorf("expected only public incident, got %+v", resp.RecentIncidents)
	}

	// 2. No public monitors -> no_monitors
	hEmpty := NewStatusHandler(&stubPublicMonitorStore{}, &stubStatusCheckStore{}, incidents)
	w = httptest.NewRecorder()
	hEmpty.PublicStatus(w, httptest.NewRequest("GET", "/api/v1/status", nil))
	var emptyResp model.PublicStatusResponse
	_ = json.NewDecoder(w.Body).Decode(&emptyResp)
	if emptyResp.OverallStatus != model.OverallStatusNoMonitors {
		t.Errorf("expected no_monitors, got %s", emptyResp.OverallStatus)
	}

	// 3. All down -> major_outage
	hDown := NewStatusHandler(
		&stubPublicMonitorStore{monitors: []model.Monitor{{ID: pub2, Name: "Checkout", Public: true, Status: "active"}}},
		&stubStatusCheckStore{
			latest: map[primitive.ObjectID]*model.CheckResult{pub2: {Success: false, CheckedAt: now}},
		},
		incidents,
	)
	w = httptest.NewRecorder()
	hDown.PublicStatus(w, httptest.NewRequest("GET", "/api/v1/status", nil))
	var downResp model.PublicStatusResponse
	_ = json.NewDecoder(w.Body).Decode(&downResp)
	if downResp.OverallStatus != model.OverallStatusMajorOutage {
		t.Errorf("expected major_outage, got %s", downResp.OverallStatus)
	}
}

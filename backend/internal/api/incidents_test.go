package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubIncidentLister struct {
	incidents []model.Incident
}

func (s *stubIncidentLister) ListByMonitor(_ context.Context, monitorID primitive.ObjectID, _, _ int) ([]model.Incident, error) {
	var out []model.Incident
	for _, inc := range s.incidents {
		if inc.MonitorID == monitorID {
			out = append(out, inc)
		}
	}
	return out, nil
}

func (s *stubIncidentLister) ListRecent(_ context.Context, status string, _, _ int) ([]model.Incident, error) {
	var out []model.Incident
	for _, inc := range s.incidents {
		if status == "" || inc.Status == status {
			out = append(out, inc)
		}
	}
	return out, nil
}

func TestIncidentHandlers(t *testing.T) {
	monID := primitive.NewObjectID()
	now := time.Now()
	resolvedAt := now.Add(2 * time.Minute)

	store := &stubIncidentLister{
		incidents: []model.Incident{
			{
				ID:          primitive.NewObjectID(),
				MonitorID:   monID,
				MonitorName: "API",
				URL:         "https://api.example.com",
				Status:      model.IncidentStatusOpen,
				Cause:       "HTTP 503",
				FailCount:   2,
				StartedAt:   now,
			},
			{
				ID:          primitive.NewObjectID(),
				MonitorID:   monID,
				MonitorName: "API",
				URL:         "https://api.example.com",
				Status:      model.IncidentStatusResolved,
				Cause:       "timeout",
				FailCount:   1,
				StartedAt:   now.Add(-10 * time.Minute),
				ResolvedAt:  &resolvedAt,
				DurationSec: 120,
			},
		},
	}

	h := NewIncidentHandler(store)
	r := chi.NewRouter()
	r.Get("/incidents", h.List)
	r.Get("/monitors/{id}/incidents", h.ListByMonitor)

	// List all
	req := httptest.NewRequest("GET", "/incidents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var all []model.Incident
	if err := json.NewDecoder(w.Body).Decode(&all); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 incidents, got %d", len(all))
	}

	// Filter by status=open
	req = httptest.NewRequest("GET", "/incidents?status=open", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var open []model.Incident
	_ = json.NewDecoder(w.Body).Decode(&open)
	if len(open) != 1 || open[0].Status != model.IncidentStatusOpen {
		t.Fatalf("expected 1 open incident, got %+v", open)
	}

	// List by monitor id
	req = httptest.NewRequest("GET", "/monitors/"+monID.Hex()+"/incidents", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Invalid monitor id -> 400
	req = httptest.NewRequest("GET", "/monitors/not-an-objectid/incidents", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid monitor id, got %d", w.Code)
	}
}

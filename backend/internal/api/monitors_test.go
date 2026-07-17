package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	// Inline handler for health
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestCreateMonitorValidation(t *testing.T) {
	handler := &MonitorHandler{
		rdb: nil,
	}

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "empty body",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing name",
			body:     `{"url":"http://example.com"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing url",
			body:     `{"name":"test"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json",
			body:     `{invalid}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/monitors", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			handler.Create(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func TestMonitorModel(t *testing.T) {
	m := &model.Monitor{
		Name:     "test",
		URL:      "https://example.com",
		Method:   "GET",
		Interval: 60,
		Timeout:  5000,
		Status:   "active",
	}

	if m.Name != "test" {
		t.Errorf("expected name test, got %s", m.Name)
	}
	if m.URL != "https://example.com" {
		t.Errorf("expected url https://example.com, got %s", m.URL)
	}
	if m.Method != "GET" {
		t.Errorf("expected method GET, got %s", m.Method)
	}
}

func TestCheckResultModel(t *testing.T) {
	cr := &model.CheckResult{
		MonitorID:  primitive.NewObjectID(),
		StatusCode: 200,
		LatencyMs:  150,
		Success:    true,
	}

	if !cr.Success {
		t.Error("expected success to be true")
	}
	if cr.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", cr.StatusCode)
	}
	if cr.LatencyMs != 150 {
		t.Errorf("expected latency 150, got %d", cr.LatencyMs)
	}
}

func TestCheckJobModel(t *testing.T) {
	job := model.CheckJob{
		MonitorID: primitive.NewObjectID().Hex(),
		URL:       "https://example.com",
		Method:    "GET",
		TimeoutMs: 5000,
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded model.CheckJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.URL != job.URL {
		t.Errorf("expected url %s, got %s", job.URL, decoded.URL)
	}
}

func TestAlertMessageModel(t *testing.T) {
	alert := model.AlertMessage{
		MonitorID:   primitive.NewObjectID().Hex(),
		MonitorName: "test",
		URL:         "https://example.com",
		FailCount:   3,
		LastError:   "timeout",
	}

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded model.AlertMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.FailCount != 3 {
		t.Errorf("expected fail count 3, got %d", decoded.FailCount)
	}
}

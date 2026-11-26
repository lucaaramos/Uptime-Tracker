package alerting

import (
	"strings"
	"testing"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
)

func TestFormatAlertDown(t *testing.T) {
	mon := &model.Monitor{Name: "Payments API", URL: "https://api.example.com/health"}
	ts := time.Date(2026, 11, 26, 15, 0, 0, 0, time.UTC).UnixMilli()

	subject, body := formatAlert(mon, model.AlertMessage{
		Type:      model.AlertTypeDown,
		FailCount: 3,
		LastError: "HTTP 503",
		Timestamp: ts,
	})

	if subject != "[Uptime Tracker] Payments API is DOWN" {
		t.Errorf("unexpected subject: %q", subject)
	}
	if !strings.Contains(body, "Fail Count: 3") || !strings.Contains(body, "Last Error: HTTP 503") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatAlertRecovery(t *testing.T) {
	mon := &model.Monitor{Name: "Payments API", URL: "https://api.example.com/health"}
	ts := time.Date(2026, 11, 26, 15, 5, 12, 0, time.UTC).UnixMilli()

	subject, body := formatAlert(mon, model.AlertMessage{
		Type:        model.AlertTypeRecovery,
		DurationSec: 312,
		Timestamp:   ts,
	})

	if subject != "[Uptime Tracker] Payments API is BACK UP" {
		t.Errorf("unexpected subject: %q", subject)
	}
	if !strings.Contains(body, "Status: Operational") || !strings.Contains(body, "Downtime: 5m 12s") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		sec  int64
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{125, "2m 5s"},
		{3661, "1h 1m 1s"},
	}
	for _, tc := range cases {
		if got := formatDuration(tc.sec); got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.sec, got, tc.want)
		}
	}
}

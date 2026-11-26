package worker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/lv99/uptime-tracker/internal/config"
	"github.com/lv99/uptime-tracker/internal/model"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.10", true},
		{"169.254.169.254", true},
		{"100.64.0.1", true},
		{"0.0.0.0", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2606:4700:4700::1111", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isPrivateIP(ip); got != tt.private {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
		}
	}
}

func TestValidateTargetURL(t *testing.T) {
	if err := validateTargetURL("https://example.com/health"); err != nil {
		t.Errorf("expected valid https url, got %v", err)
	}
	if err := validateTargetURL("http://example.com"); err != nil {
		t.Errorf("expected valid http url, got %v", err)
	}
	if err := validateTargetURL("file:///etc/passwd"); err == nil {
		t.Error("expected file:// scheme to be rejected")
	}
	if err := validateTargetURL("http://"); err == nil {
		t.Error("expected empty host to be rejected")
	}
}

func TestSafeDialContextBlocksPrivateIPs(t *testing.T) {
	blockedAddrs := []string{
		"127.0.0.1:8080",
		"[::1]:8080",
		"169.254.169.254:80",
		"10.0.0.1:443",
		"192.168.1.1:80",
	}

	for _, addr := range blockedAddrs {
		conn, err := safeDialContext(context.Background(), "tcp", addr)
		if conn != nil {
			conn.Close()
		}
		if !errors.Is(err, ErrBlockedAddress) {
			t.Errorf("safeDialContext(%s): expected ErrBlockedAddress, got %v", addr, err)
		}
	}
}

func TestExecuteCheckSSRFProtection(t *testing.T) {
	// Blocked when AllowPrivateURLs is false
	wBlocked := New(&config.Config{AllowPrivateURLs: false}, nil, nil, nil, nil)
	resBlocked := wBlocked.executeCheck(model.CheckJob{
		URL:       "http://127.0.0.1:18080/health",
		Method:    "GET",
		TimeoutMs: 2000,
	})
	if resBlocked.Success {
		t.Fatal("expected loopback check to be blocked when AllowPrivateURLs=false")
	}
	if !strings.Contains(resBlocked.ErrorMsg, ErrBlockedAddress.Error()) {
		t.Errorf("expected SSRF blocked error, got %q", resBlocked.ErrorMsg)
	}

	// Blocked scheme
	resScheme := wBlocked.executeCheck(model.CheckJob{
		URL:       "file:///etc/passwd",
		Method:    "GET",
		TimeoutMs: 2000,
	})
	if resScheme.Success || !strings.Contains(resScheme.ErrorMsg, "unsupported url scheme") {
		t.Errorf("expected unsupported scheme error, got success=%v err=%q", resScheme.Success, resScheme.ErrorMsg)
	}

	// When AllowPrivateURLs is true, custom SSRF DialContext is not installed
	wAllowed := New(&config.Config{AllowPrivateURLs: true}, nil, nil, nil, nil)
	tr, ok := wAllowed.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.DialContext != nil {
		t.Fatal("expected DialContext to be nil when AllowPrivateURLs=true")
	}
}

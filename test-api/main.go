package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ok", handleOK)
	mux.HandleFunc("/slow", handleSlow)
	mux.HandleFunc("/flaky", handleFlaky)
	mux.HandleFunc("/error", handleError)
	mux.HandleFunc("/status/", handleStatus)
	mux.HandleFunc("/echo", handleEcho)

	addr := ":" + port
	log.Printf("test-api listening on %s", addr)
	log.Printf("  /health  → always 200")
	log.Printf("  /ok      → always 200")
	log.Printf("  /slow    → 200 after N ms (default: 2000)")
	log.Printf("  /flaky   → 200/500 randomly")
	log.Printf("  /error   → always 500")
	log.Printf("  /status/N → returns status code N")
	log.Printf("  /echo    → echoes request info as JSON")
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "test-api",
		"version": "1.0.0",
		"endpoints": []string{
			"/health", "/ok", "/slow", "/flaky", "/error", "/status/{code}", "/echo",
		},
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func handleOK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "everything is fine",
	})
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	delayStr := r.URL.Query().Get("ms")
	delay := 2000
	if delayStr != "" {
		if d, err := strconv.Atoi(delayStr); err == nil && d > 0 && d <= 30000 {
			delay = d
		}
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"delay_ms":    delay,
		"message":     fmt.Sprintf("delayed response after %dms", delay),
	})
}

func handleFlaky(w http.ResponseWriter, r *http.Request) {
	rateStr := r.URL.Query().Get("rate")
	failRate := 0.3
	if rateStr != "" {
		if f, err := strconv.ParseFloat(rateStr, 64); err == nil && f >= 0 && f <= 1 {
			failRate = f
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if rand.Float64() < failRate {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "random failure",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "lucky you",
	})
}

func handleError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"message": "this endpoint always fails",
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Path[len("/status/"):]
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		code = 400
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      http.StatusText(code),
		"status_code": code,
	})
}

func handleEcho(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]interface{}{
		"method":     r.Method,
		"url":        r.URL.String(),
		"host":       r.Host,
		"remote":     r.RemoteAddr,
		"user_agent": r.UserAgent(),
		"headers":    r.Header,
		"timestamp":  time.Now().UnixMilli(),
	}

	json.NewEncoder(w).Encode(info)
}

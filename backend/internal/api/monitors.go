package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MonitorHandler struct {
	monitor *repository.MonitorRepo
	check   *repository.CheckRepo
	rdb     *redis.Client
}

func NewMonitorHandler(monitor *repository.MonitorRepo, check *repository.CheckRepo, rdb *redis.Client) *MonitorHandler {
	return &MonitorHandler{monitor: monitor, check: check, rdb: rdb}
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.MonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and url are required"})
		return
	}

	m := &model.Monitor{
		Name:         req.Name,
		URL:          req.URL,
		Method:       req.Method,
		Interval:     req.Interval,
		Timeout:      req.Timeout,
		Public:       req.Public,
		Headers:      req.Headers,
		AlertEmail:   req.AlertEmail,
		AlertWebhook: req.AlertWebhook,
	}

	if err := h.monitor.Create(r.Context(), m); err != nil {
		log.Error().Err(err).Msg("failed to create monitor")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create monitor"})
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.monitor.List(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list monitors")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list monitors"})
		return
	}

	responses := make([]model.MonitorResponse, 0, len(monitors))
	for _, m := range monitors {
		resp := model.MonitorResponse{Monitor: m}

		// Get last result from cache or DB
		cacheKey := fmt.Sprintf("monitor:%s:last_result", m.ID.Hex())
		cached, err := h.rdb.Get(r.Context(), cacheKey).Result()
		if err == nil && cached != "" {
			var last model.CheckResult
			if json.Unmarshal([]byte(cached), &last) == nil {
				resp.LastResult = &last
			}
		}
		if resp.LastResult == nil {
			last, err := h.check.GetLatestByMonitorID(r.Context(), m.ID)
			if err == nil {
				resp.LastResult = last
			}
		}

		// 24h uptime
		stats, err := h.check.GetStats(r.Context(), m.ID, time.Now().Add(-24*time.Hour))
		if err == nil {
			resp.Uptime24h = stats.UptimePercent
		}

		responses = append(responses, resp)
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h *MonitorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	m, err := h.monitor.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "monitor not found"})
		return
	}

	resp := model.MonitorResponse{Monitor: *m}

	last, err := h.check.GetLatestByMonitorID(r.Context(), id)
	if err == nil {
		resp.LastResult = last
	}

	stats, err := h.check.GetStats(r.Context(), id, time.Now().Add(-24*time.Hour))
	if err == nil {
		resp.Uptime24h = stats.UptimePercent
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	var req model.MonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.monitor.Update(r.Context(), id, req); err != nil {
		log.Error().Err(err).Msg("failed to update monitor")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update monitor"})
		return
	}

	m, err := h.monitor.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch updated monitor"})
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	if err := h.monitor.Delete(r.Context(), id); err != nil {
		log.Error().Err(err).Msg("failed to delete monitor")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete monitor"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *MonitorHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	if err := h.monitor.SetStatus(r.Context(), id, "paused"); err != nil {
		log.Error().Err(err).Msg("failed to pause monitor")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to pause monitor"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (h *MonitorHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	if err := h.monitor.SetStatus(r.Context(), id, "active"); err != nil {
		log.Error().Err(err).Msg("failed to resume monitor")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resume monitor"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (h *MonitorHandler) History(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	results, err := h.check.GetByMonitorID(r.Context(), id, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to get history")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get history"})
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *MonitorHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	duration := 24 * time.Hour
	switch r.URL.Query().Get("period") {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	}

	stats, err := h.check.GetStats(r.Context(), id, time.Now().Add(-duration))
	if err != nil {
		log.Error().Err(err).Msg("failed to get stats")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IncidentLister interface {
	ListByMonitor(ctx context.Context, monitorID primitive.ObjectID, limit, offset int) ([]model.Incident, error)
	ListRecent(ctx context.Context, status string, limit, offset int) ([]model.Incident, error)
}

type IncidentHandler struct {
	incidents IncidentLister
}

func NewIncidentHandler(incidents IncidentLister) *IncidentHandler {
	return &IncidentHandler{incidents: incidents}
}

func parseLimitOffset(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)
	status := r.URL.Query().Get("status")

	incidents, err := h.incidents.ListRecent(r.Context(), status, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list incidents")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list incidents"})
		return
	}
	if incidents == nil {
		incidents = []model.Incident{}
	}

	writeJSON(w, http.StatusOK, incidents)
}

func (h *IncidentHandler) ListByMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid monitor id"})
		return
	}

	limit, offset := parseLimitOffset(r)
	incidents, err := h.incidents.ListByMonitor(r.Context(), id, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list monitor incidents")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list incidents"})
		return
	}
	if incidents == nil {
		incidents = []model.Incident{}
	}

	writeJSON(w, http.StatusOK, incidents)
}

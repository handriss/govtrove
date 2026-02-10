package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/repository"
)

type AnalyticsHandler struct {
	repo   *repository.AnalyticsRepository
	logger *slog.Logger
}

func NewAnalyticsHandler(repo *repository.AnalyticsRepository, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo, logger: logger}
}

func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}

	validPeriods := map[string]bool{"24h": true, "7d": true, "30d": true}
	if !validPeriods[period] {
		http.Error(w, "invalid period (use 24h, 7d, or 30d)", http.StatusBadRequest)
		return
	}

	analytics, err := h.repo.GetSearchAnalytics(r.Context(), period)
	if err != nil {
		h.logger.Error("get analytics failed", "period", period, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, analytics)
}

func (h *AnalyticsHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

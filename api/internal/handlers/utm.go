package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/handriss/govtrove/api/internal/repository"
)

type UTMHandler struct {
	repo   *repository.UTMRepository
	logger *slog.Logger
}

func NewUTMHandler(repo *repository.UTMRepository, logger *slog.Logger) *UTMHandler {
	return &UTMHandler{repo: repo, logger: logger}
}

type utmVisitRequest struct {
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	LandingPage string  `json:"landing_page"`
	Origin      string  `json:"origin"`
	Referrer    *string `json:"referrer"`
}

func truncatePtr(s *string, max int) *string {
	if s == nil {
		return nil
	}
	v := *s
	if len(v) > max {
		v = v[:max]
	}
	return &v
}

func (h *UTMHandler) TrackVisit(w http.ResponseWriter, r *http.Request) {
	var req utmVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.UTMSource == nil && req.UTMMedium == nil && req.UTMCampaign == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.LandingPage == "" {
		req.LandingPage = "/"
	}

	ua := r.Header.Get("User-Agent")
	var uaPtr *string
	if ua != "" {
		if len(ua) > 512 {
			ua = ua[:512]
		}
		uaPtr = &ua
	}

	origin := req.Origin
	if origin != "landing" && origin != "app" {
		origin = "unknown"
	}

	visit := &repository.UTMVisit{
		UTMSource:   truncatePtr(req.UTMSource, 255),
		UTMMedium:   truncatePtr(req.UTMMedium, 255),
		UTMCampaign: truncatePtr(req.UTMCampaign, 255),
		LandingPage: req.LandingPage,
		Origin:      &origin,
		Referrer:    truncatePtr(req.Referrer, 2048),
		UserAgent:   uaPtr,
	}

	if err := h.repo.CreateVisit(r.Context(), visit); err != nil {
		h.logger.Error("failed to track utm visit", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UTMHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	var since *time.Time
	now := time.Now()
	switch period {
	case "7d":
		t := now.AddDate(0, 0, -7)
		since = &t
	case "90d":
		t := now.AddDate(0, 0, -90)
		since = &t
	case "all":
		since = nil
	default:
		t := now.AddDate(0, 0, -30)
		since = &t
	}

	analytics, err := h.repo.GetAnalytics(r.Context(), since)
	if err != nil {
		h.logger.Error("failed to get utm analytics", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

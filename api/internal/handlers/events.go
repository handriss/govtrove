package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

type EventHandler struct {
	repo   *repository.EventRepository
	logger *slog.Logger
}

func NewEventHandler(repo *repository.EventRepository, logger *slog.Logger) *EventHandler {
	return &EventHandler{repo: repo, logger: logger}
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}

	validTypes := map[string]bool{"search": true, "filter": true, "click": true, "page": true}
	if !validTypes[req.EventType] {
		http.Error(w, "invalid event_type", http.StatusBadRequest)
		return
	}

	if req.Filters != nil {
		filtersJSON, err := json.Marshal(req.Filters)
		if err != nil || len(filtersJSON) > 4*1024 {
			http.Error(w, "Filters too large", http.StatusBadRequest)
			return
		}
	}

	req.SessionID = truncateStr(req.SessionID, 64)
	req.Query = truncateStr(req.Query, 500)
	req.SortBy = truncateStr(req.SortBy, 32)

	event := &models.SearchEvent{
		EventType: req.EventType,
		Filters:   req.Filters,
	}

	if req.SessionID != "" {
		event.SessionID = &req.SessionID
	}
	if req.Query != "" {
		event.Query = &req.Query
	}
	if req.SortBy != "" {
		event.SortBy = &req.SortBy
	}
	if req.Page > 0 {
		event.Page = &req.Page
	}
	if req.TotalResults > 0 {
		event.TotalResults = &req.TotalResults
	}
	if req.ResultPosition > 0 {
		event.ResultPosition = &req.ResultPosition
	}
	if req.OpportunityID > 0 {
		event.OpportunityID = &req.OpportunityID
	}

	userAgent := truncateStr(r.Header.Get("User-Agent"), 512)
	if userAgent != "" {
		event.UserAgent = &userAgent
	}

	referer := truncateStr(r.Header.Get("Referer"), 2048)
	if referer != "" {
		event.Referer = &referer
	}

	if err := h.repo.Create(r.Context(), event); err != nil {
		h.logger.Error("create event failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

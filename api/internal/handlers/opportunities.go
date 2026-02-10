package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type OpportunityHandler struct {
	repo   *repository.OpportunityRepository
	logger *slog.Logger
}

func NewOpportunityHandler(repo *repository.OpportunityRepository, logger *slog.Logger) *OpportunityHandler {
	return &OpportunityHandler{repo: repo, logger: logger}
}

func (h *OpportunityHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)

	result, err := h.repo.Search(r.Context(), params)
	if err != nil {
		h.logger.Error("search failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *OpportunityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	opp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("get opportunity failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if opp == nil {
		http.Error(w, "Opportunity not found", http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, opp)
}

func (h *OpportunityHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	options, err := h.repo.GetFilterOptions(r.Context())
	if err != nil {
		h.logger.Error("get filters failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, options)
}

func (h *OpportunityHandler) parseSearchParams(r *http.Request) models.SearchParams {
	q := r.URL.Query()

	params := models.SearchParams{
		Query:  q.Get("q"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
		Page:   1,
		Limit:  25,
	}

	if typeStr := q.Get("type"); typeStr != "" {
		params.Types = strings.Split(typeStr, ",")
	}

	if setAsideStr := q.Get("set_aside"); setAsideStr != "" {
		params.SetAsides = strings.Split(setAsideStr, ",")
	}

	if naicsStr := q.Get("naics"); naicsStr != "" {
		params.NAICSCodes = strings.Split(naicsStr, ",")
	}

	if stateStr := q.Get("state"); stateStr != "" {
		params.States = strings.Split(stateStr, ",")
	}

	if postedFrom := q.Get("posted_from"); postedFrom != "" {
		if t, err := time.Parse("2006-01-02", postedFrom); err == nil {
			params.PostedFrom = &t
		}
	}

	if postedTo := q.Get("posted_to"); postedTo != "" {
		if t, err := time.Parse("2006-01-02", postedTo); err == nil {
			params.PostedTo = &t
		}
	}

	if deadlineFrom := q.Get("deadline_from"); deadlineFrom != "" {
		if t, err := time.Parse("2006-01-02", deadlineFrom); err == nil {
			params.DeadlineFrom = &t
		}
	}

	if deadlineTo := q.Get("deadline_to"); deadlineTo != "" {
		if t, err := time.Parse("2006-01-02", deadlineTo); err == nil {
			params.DeadlineTo = &t
		}
	}

	if pageStr := q.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}

	return params
}

func (h *OpportunityHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

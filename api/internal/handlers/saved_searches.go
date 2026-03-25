package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type SavedSearchHandler struct {
	repo     *repository.SavedSearchRepository
	userRepo *repository.UserRepository
	logger   *slog.Logger
	eventLog *EventLogger
}

func NewSavedSearchHandler(
	repo *repository.SavedSearchRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
	eventLog *EventLogger,
) *SavedSearchHandler {
	return &SavedSearchHandler{repo: repo, userRepo: userRepo, logger: logger, eventLog: eventLog}
}

func (h *SavedSearchHandler) resolveUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
	workosID := authmw.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil {
		h.logger.Error("failed to look up user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return 0, false
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return 0, false
	}
	return user.ID, true
}

func (h *SavedSearchHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	searches, err := h.repo.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list saved searches", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if searches == nil {
		searches = []models.SavedSearch{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searches)
}

func (h *SavedSearchHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	var input models.CreateSavedSearchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) > 200 {
		input.Name = input.Name[:200]
	}

	if input.Name == "" || len(input.Filters) == 0 {
		http.Error(w, "Name and filters are required", http.StatusBadRequest)
		return
	}
	if len(input.Filters) > 10*1024 {
		http.Error(w, "Filters too large", http.StatusBadRequest)
		return
	}

	search, err := h.repo.Create(r.Context(), userID, &input)
	if err != nil {
		h.logger.Error("failed to create saved search", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(search)

	var filters map[string]interface{}
	if err := json.Unmarshal(input.Filters, &filters); err == nil {
		h.eventLog.Log(r, &userID, &models.SearchEvent{
			EventType: "save_search",
			Filters:   filters,
		})
	}
}

func (h *SavedSearchHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var input models.UpdateSavedSearchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			http.Error(w, "Name cannot be empty", http.StatusBadRequest)
			return
		}
		if len(trimmed) > 200 {
			trimmed = trimmed[:200]
		}
		input.Name = &trimmed
	}
	if input.Filters != nil && len(*input.Filters) > 10*1024 {
		http.Error(w, "Filters too large", http.StatusBadRequest)
		return
	}

	search, err := h.repo.Update(r.Context(), id, userID, &input)
	if err != nil {
		h.logger.Error("failed to update saved search", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if search == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(search)
}

func (h *SavedSearchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id, userID); err != nil {
		h.logger.Error("failed to delete saved search", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.eventLog.Log(r, &userID, &models.SearchEvent{
		EventType: "delete_search",
	})
}

func (h *SavedSearchHandler) HistoryTimeline(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	days := 7
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	if days > 30 {
		days = 30
	}

	result, err := h.repo.HistoryTimeline(r.Context(), id, userID, days)
	if err != nil {
		h.logger.Error("failed to get history timeline", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *SavedSearchHandler) HistoryDay(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" || len(date) != 10 {
		http.Error(w, "date parameter required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 25
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	search, result, err := h.repo.HistoryDay(r.Context(), id, userID, date, page, limit)
	if err != nil {
		h.logger.Error("failed to get history day", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if search == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *SavedSearchHandler) Run(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	search, result, err := h.repo.Run(r.Context(), id, userID)
	if err != nil {
		h.logger.Error("failed to run saved search", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if search == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"search":  search,
		"results": result,
	})
}

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type SavedOpportunityHandler struct {
	repo     *repository.SavedOpportunityRepository
	userRepo *repository.UserRepository
	logger   *slog.Logger
}

func NewSavedOpportunityHandler(
	repo *repository.SavedOpportunityRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
) *SavedOpportunityHandler {
	return &SavedOpportunityHandler{repo: repo, userRepo: userRepo, logger: logger}
}

func (h *SavedOpportunityHandler) resolveUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
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

func (h *SavedOpportunityHandler) ListIDs(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	ids, err := h.repo.ListIDs(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list saved opportunity ids", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if ids == nil {
		ids = []int{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=30")
	json.NewEncoder(w).Encode(map[string][]int{"opportunity_ids": ids})
}

func (h *SavedOpportunityHandler) ListWithDetails(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	sort := r.URL.Query().Get("sort")
	activeOnly := r.URL.Query().Get("active_only") == "true"

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 25
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	details, total, err := h.repo.ListWithDetails(r.Context(), userID, sort, activeOnly, page, limit)
	if err != nil {
		h.logger.Error("failed to list saved opportunities with details", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if details == nil {
		details = []models.SavedOpportunityDetail{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"opportunities": details,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

func (h *SavedOpportunityHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		OpportunityID int `json:"opportunity_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OpportunityID <= 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Add(r.Context(), userID, req.OpportunityID); err != nil {
		h.logger.Error("failed to save opportunity", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *SavedOpportunityHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		OpportunityID int `json:"opportunity_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OpportunityID <= 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Remove(r.Context(), userID, req.OpportunityID); err != nil {
		h.logger.Error("failed to remove saved opportunity", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SavedOpportunityHandler) RemoveByOpportunityID(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	opportunityID, err := strconv.Atoi(chi.URLParam(r, "opportunityId"))
	if err != nil || opportunityID <= 0 {
		http.Error(w, "Invalid opportunity ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.RemoveByOpportunityID(r.Context(), userID, opportunityID); err != nil {
		h.logger.Error("failed to remove saved opportunity by opportunity id", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SavedOpportunityHandler) BulkAdd(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		OpportunityIDs []int `json:"opportunity_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.OpportunityIDs) > 100 {
		http.Error(w, "Maximum 100 opportunities per request", http.StatusBadRequest)
		return
	}

	valid := req.OpportunityIDs[:0]
	for _, id := range req.OpportunityIDs {
		if id > 0 {
			valid = append(valid, id)
		}
	}
	req.OpportunityIDs = valid

	if len(req.OpportunityIDs) == 0 {
		w.WriteHeader(http.StatusCreated)
		return
	}

	if err := h.repo.BulkAdd(r.Context(), userID, req.OpportunityIDs); err != nil {
		h.logger.Error("failed to bulk save opportunities", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *SavedOpportunityHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Notes) > 5000 {
		http.Error(w, "Notes too long", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateNotes(r.Context(), id, userID, req.Notes); err != nil {
		h.logger.Error("failed to update notes", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

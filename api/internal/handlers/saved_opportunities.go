package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	authmw "github.com/handriss/govtrove/api/internal/middleware"
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

func (h *SavedOpportunityHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	ids, err := h.repo.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list saved opportunities", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if ids == nil {
		ids = []int{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]int{"opportunity_ids": ids})
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

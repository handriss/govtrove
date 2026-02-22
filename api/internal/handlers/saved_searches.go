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

type SavedSearchHandler struct {
	repo     *repository.SavedSearchRepository
	userRepo *repository.UserRepository
	logger   *slog.Logger
}

func NewSavedSearchHandler(
	repo *repository.SavedSearchRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
) *SavedSearchHandler {
	return &SavedSearchHandler{repo: repo, userRepo: userRepo, logger: logger}
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
	if input.Name == "" || len(input.Filters) == 0 {
		http.Error(w, "Name and filters are required", http.StatusBadRequest)
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
}

func (h *SavedSearchHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var input models.UpdateSavedSearchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id, userID); err != nil {
		h.logger.Error("failed to delete saved search", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

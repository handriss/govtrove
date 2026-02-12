package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type AuthHandler struct {
	repo   *repository.UserRepository
	logger *slog.Logger
}

func NewAuthHandler(repo *repository.UserRepository, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{repo: repo, logger: logger}
}

type syncRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (h *AuthHandler) Sync(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	user, err := h.repo.Upsert(r.Context(), &models.UpsertUserInput{
		WorkOSID:  workosID,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		h.logger.Error("sync user failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("user synced", "workos_id", workosID, "user_id", user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

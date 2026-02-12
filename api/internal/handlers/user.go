package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type UserHandler struct {
	repo   *repository.UserRepository
	logger *slog.Logger
}

func NewUserHandler(repo *repository.UserRepository, logger *slog.Logger) *UserHandler {
	return &UserHandler{repo: repo, logger: logger}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.repo.GetByWorkOSID(r.Context(), workosID)
	if err != nil {
		h.logger.Error("get user failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

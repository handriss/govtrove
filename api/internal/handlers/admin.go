package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/handriss/govtrove/api/internal/repository"
)

type AdminHandler struct {
	userRepo       *repository.UserRepository
	userUpdateRepo *repository.UserUpdateRepository
	logger         *slog.Logger
}

func NewAdminHandler(userRepo *repository.UserRepository, userUpdateRepo *repository.UserUpdateRepository, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, userUpdateRepo: userUpdateRepo, logger: logger}
}

type adminUserResponse struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Plan      string `json:"plan"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("list users failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	out := make([]adminUserResponse, len(users))
	for i, u := range users {
		out[i] = adminUserResponse{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Plan:      u.Plan,
			IsAdmin:   u.IsAdmin,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *AdminHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	updates, total, err := h.userUpdateRepo.List(r.Context(), userID, false, page, 50)
	if err != nil {
		h.logger.Error("admin list notifications failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"updates": updates,
		"total":   total,
		"page":    page,
		"limit":   50,
	})
}

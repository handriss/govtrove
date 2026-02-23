package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type UserUpdateHandler struct {
	repo     *repository.UserUpdateRepository
	userRepo *repository.UserRepository
	logger   *slog.Logger
}

func NewUserUpdateHandler(
	repo *repository.UserUpdateRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
) *UserUpdateHandler {
	return &UserUpdateHandler{repo: repo, userRepo: userRepo, logger: logger}
}

func (h *UserUpdateHandler) resolveUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
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

func (h *UserUpdateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 25
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	updates, total, err := h.repo.List(r.Context(), userID, unreadOnly, page, limit)
	if err != nil {
		h.logger.Error("failed to list user updates", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if updates == nil {
		updates = []models.UserUpdate{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"updates": updates,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func (h *UserUpdateHandler) Count(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	count, err := h.repo.Count(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to count user updates", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=30")
	json.NewEncoder(w).Encode(count)
}

func (h *UserUpdateHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.MarkRead(r.Context(), id, userID); err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to mark update read", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserUpdateHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	marked, err := h.repo.MarkAllRead(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to mark all updates read", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"marked": marked})
}

func (h *UserUpdateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUserID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id, userID); err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete user update", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

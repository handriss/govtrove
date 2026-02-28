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
	pipelineRepo   *repository.PipelineRepository
	logger         *slog.Logger
}

func NewAdminHandler(userRepo *repository.UserRepository, userUpdateRepo *repository.UserUpdateRepository, pipelineRepo *repository.PipelineRepository, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, userUpdateRepo: userUpdateRepo, pipelineRepo: pipelineRepo, logger: logger}
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

func (h *AdminHandler) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.pipelineRepo.ListApiKeys(r.Context())
	if err != nil {
		h.logger.Error("list api keys failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (h *AdminHandler) ListSamgovRequests(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	requests, total, err := h.pipelineRepo.ListSamgovRequests(r.Context(), page, 50)
	if err != nil {
		h.logger.Error("list samgov requests failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"requests": requests,
		"total":    total,
		"page":     page,
		"limit":    50,
	})
}

func (h *AdminHandler) ListPipelineRuns(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	fullOnly := r.URL.Query().Get("full_only") == "true"

	runs, total, err := h.pipelineRepo.ListPipelineRuns(r.Context(), page, 20, fullOnly)
	if err != nil {
		h.logger.Error("list pipeline runs failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": 20,
	})
}

func (h *AdminHandler) ListSearchEvents(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	emptyOnly := r.URL.Query().Get("empty_only") == "true"

	events, total, err := h.pipelineRepo.ListSearchEvents(r.Context(), page, 50, emptyOnly)
	if err != nil {
		h.logger.Error("list search events failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  50,
	})
}

func (h *AdminHandler) GetApiKeyUsage(w http.ResponseWriter, r *http.Request) {
	keyHash := r.URL.Query().Get("key_hash")
	if keyHash == "" {
		http.Error(w, "key_hash is required", http.StatusBadRequest)
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 90 {
			days = v
		}
	}

	buckets, err := h.pipelineRepo.GetApiKeyUsage(r.Context(), keyHash, days)
	if err != nil {
		h.logger.Error("get api key usage failed", "key_hash", keyHash, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"buckets": buckets,
	})
}

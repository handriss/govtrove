package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type AuthHandler struct {
	repo           *repository.UserRepository
	emailPrefsRepo *repository.EmailPreferencesRepository
	logger         *slog.Logger
}

func NewAuthHandler(repo *repository.UserRepository, emailPrefsRepo *repository.EmailPreferencesRepository, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{repo: repo, emailPrefsRepo: emailPrefsRepo, logger: logger}
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

	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if req.Email == "" || len(req.Email) > 320 || !emailRegex.MatchString(req.Email) {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}
	if len(req.FirstName) > 200 {
		req.FirstName = req.FirstName[:200]
	}
	if len(req.LastName) > 200 {
		req.LastName = req.LastName[:200]
	}

	result, err := h.repo.Upsert(r.Context(), &models.UpsertUserInput{
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

	if result.IsNew {
		if err := h.emailPrefsRepo.CreateDefaults(r.Context(), result.User.ID); err != nil {
			h.logger.Error("failed to create email preferences", "error", err, "user_id", result.User.ID)
		}
	}

	h.logger.Info("user synced", "workos_id", workosID, "user_id", result.User.ID, "is_new", result.IsNew)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.User)
}

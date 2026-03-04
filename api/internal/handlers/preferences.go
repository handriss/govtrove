package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type PreferencesHandler struct {
	emailPrefsRepo *repository.EmailPreferencesRepository
	userRepo       *repository.UserRepository
	logger         *slog.Logger
}

func NewPreferencesHandler(
	emailPrefsRepo *repository.EmailPreferencesRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
) *PreferencesHandler {
	return &PreferencesHandler{
		emailPrefsRepo: emailPrefsRepo,
		userRepo:       userRepo,
		logger:         logger,
	}
}

type emailPrefsResponse struct {
	SearchAlerts      bool `json:"search_alerts"`
	OpportunityAlerts bool `json:"opportunity_alerts"`
}

func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	prefs, err := h.emailPrefsRepo.GetByUserID(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("get email preferences failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := emailPrefsResponse{SearchAlerts: true, OpportunityAlerts: true}
	if prefs != nil {
		resp.SearchAlerts = prefs.SearchAlerts
		resp.OpportunityAlerts = prefs.OpportunityAlerts
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *PreferencesHandler) Update(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	var body struct {
		SearchAlerts      bool `json:"search_alerts"`
		OpportunityAlerts bool `json:"opportunity_alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.emailPrefsRepo.Upsert(r.Context(), user.ID, body.SearchAlerts, body.OpportunityAlerts); err != nil {
		h.logger.Error("update email preferences failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

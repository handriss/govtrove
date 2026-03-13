package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type GiftHandler struct {
	giftCodeRepo *repository.GiftCodeRepository
	userRepo     *repository.UserRepository
	logger       *slog.Logger
}

func NewGiftHandler(
	giftCodeRepo *repository.GiftCodeRepository,
	userRepo *repository.UserRepository,
	logger *slog.Logger,
) *GiftHandler {
	return &GiftHandler{
		giftCodeRepo: giftCodeRepo,
		userRepo:     userRepo,
		logger:       logger,
	}
}

func (h *GiftHandler) Redeem(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	gc, err := h.giftCodeRepo.GetByCode(r.Context(), body.Code)
	if err != nil {
		h.logger.Error("gift code lookup failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if gc == nil {
		http.Error(w, "Invalid gift code", http.StatusNotFound)
		return
	}

	if gc.DeactivatedAt != nil {
		http.Error(w, "This gift code has been deactivated", http.StatusGone)
		return
	}
	if gc.ExpiresAt != nil && gc.ExpiresAt.Before(time.Now()) {
		http.Error(w, "This gift code has expired", http.StatusGone)
		return
	}
	if gc.MaxRedemptions > 0 && gc.RedemptionCount >= gc.MaxRedemptions {
		http.Error(w, "This gift code has reached its maximum number of redemptions", http.StatusGone)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	already, err := h.giftCodeRepo.HasUserRedeemed(r.Context(), gc.ID, user.ID)
	if err != nil {
		h.logger.Error("check redemption failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if already {
		http.Error(w, "You have already redeemed this gift code", http.StatusConflict)
		return
	}

	grantedUntil := time.Now().Add(time.Duration(gc.DurationDays) * 24 * time.Hour)

	if err := h.giftCodeRepo.AddRedemption(r.Context(), gc.ID, user.ID, grantedUntil); err != nil {
		h.logger.Error("add gift redemption failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.SetGiftExpiry(r.Context(), user.ID, &grantedUntil); err != nil {
		h.logger.Error("set gift expiry failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("gift code redeemed", "user_id", user.ID, "code", gc.Code, "granted_until", grantedUntil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"granted_until": grantedUntil,
	})
}

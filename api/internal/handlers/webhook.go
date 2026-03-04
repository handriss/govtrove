package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/email"
	"github.com/handriss/govtrove/api/internal/repository"
)

type WebhookHandler struct {
	sentEmailsRepo *repository.SentEmailsRepository
	emailPrefsRepo *repository.EmailPreferencesRepository
	emailSvc       *email.Service
	logger         *slog.Logger
}

func NewWebhookHandler(
	sentEmailsRepo *repository.SentEmailsRepository,
	emailPrefsRepo *repository.EmailPreferencesRepository,
	emailSvc *email.Service,
	logger *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		sentEmailsRepo: sentEmailsRepo,
		emailPrefsRepo: emailPrefsRepo,
		emailSvc:       emailSvc,
		logger:         logger,
	}
}

type resendWebhookEvent struct {
	Type string `json:"type"`
	Data struct {
		EmailID string `json:"email_id"`
	} `json:"data"`
}

func (h *WebhookHandler) HandleResend(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	svixID := r.Header.Get("svix-id")
	svixTimestamp := r.Header.Get("svix-timestamp")
	svixSignature := r.Header.Get("svix-signature")
	if h.emailSvc != nil && svixSignature != "" {
		if !h.emailSvc.VerifyWebhookSignature(body, svixID, svixTimestamp, svixSignature) {
			h.logger.Warn("invalid webhook signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	var event resendWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("failed to parse webhook", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	resendID := event.Data.EmailID
	if resendID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()

	switch event.Type {
	case "email.delivered":
		if err := h.sentEmailsRepo.UpdateStatusByResendID(ctx, resendID, "delivered"); err != nil {
			h.logger.Error("failed to update email status", "resend_id", resendID, "error", err)
		}

	case "email.bounced":
		if err := h.sentEmailsRepo.UpdateStatusByResendID(ctx, resendID, "bounced"); err != nil {
			h.logger.Error("failed to update email status", "resend_id", resendID, "error", err)
		}
		userID, _ := h.sentEmailsRepo.GetUserIDByResendID(ctx, resendID)
		if userID > 0 {
			if err := h.emailPrefsRepo.Unsubscribe(ctx, userID, "bounce"); err != nil {
				h.logger.Error("failed to unsubscribe bounced user", "user_id", userID, "error", err)
			}
		}

	case "email.complained":
		if err := h.sentEmailsRepo.UpdateStatusByResendID(ctx, resendID, "complained"); err != nil {
			h.logger.Error("failed to update email status", "resend_id", resendID, "error", err)
		}
		userID, _ := h.sentEmailsRepo.GetUserIDByResendID(ctx, resendID)
		if userID > 0 {
			if err := h.emailPrefsRepo.Unsubscribe(ctx, userID, "complaint"); err != nil {
				h.logger.Error("failed to unsubscribe complaining user", "user_id", userID, "error", err)
			}
		}

	default:
		h.logger.Debug("unhandled webhook event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

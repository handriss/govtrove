package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/handriss/govtrove/api/internal/email"
	"github.com/handriss/govtrove/api/internal/repository"
)

type UnsubscribeHandler struct {
	emailPrefsRepo *repository.EmailPreferencesRepository
	emailSvc       *email.Service
	logger         *slog.Logger
}

func NewUnsubscribeHandler(
	emailPrefsRepo *repository.EmailPreferencesRepository,
	emailSvc *email.Service,
	logger *slog.Logger,
) *UnsubscribeHandler {
	return &UnsubscribeHandler{
		emailPrefsRepo: emailPrefsRepo,
		emailSvc:       emailSvc,
		logger:         logger,
	}
}

func (h *UnsubscribeHandler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	uidStr := r.URL.Query().Get("uid")

	if token == "" || uidStr == "" {
		http.Error(w, "invalid unsubscribe link", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(uidStr)
	if err != nil {
		http.Error(w, "invalid unsubscribe link", http.StatusBadRequest)
		return
	}

	if h.emailSvc == nil || !h.emailSvc.ValidateUnsubscribeToken(userID, token) {
		http.Error(w, "invalid or expired unsubscribe link", http.StatusBadRequest)
		return
	}

	if err := h.emailPrefsRepo.Unsubscribe(r.Context(), userID, "user"); err != nil {
		h.logger.Error("failed to unsubscribe user", "user_id", userID, "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	h.logger.Info("user unsubscribed via email link", "user_id", userID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>Unsubscribed</title></head>
<body style="margin:0;padding:60px 20px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;text-align:center;background:#f9fafb;">
<div style="max-width:400px;margin:0 auto;">
<h1 style="font-size:22px;font-weight:700;color:#111827;margin-bottom:12px;">GovTrove</h1>
<p style="font-size:16px;color:#374151;line-height:1.6;">You've been unsubscribed from GovTrove emails.</p>
<p style="font-size:14px;color:#6b7280;margin-top:24px;">You can re-enable email notifications anytime from your <a href="https://app.govtrove.com/profile" style="color:#2563eb;">profile settings</a>.</p>
</div>
</body>
</html>`))
}

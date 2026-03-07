package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v82"

	"github.com/handriss/govtrove/api/internal/email"
	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type AdminHandler struct {
	userRepo         *repository.UserRepository
	notificationRepo *repository.NotificationRepository
	pipelineRepo     *repository.PipelineRepository
	emailPrefsRepo   *repository.EmailPreferencesRepository
	sentEmailsRepo   *repository.SentEmailsRepository
	promoRepo        *repository.PromoCodeRepository
	emailSvc         *email.Service
	sc               *stripe.Client
	promoCouponID    string
	appURL           string
	workosAPIKey     string
	logger           *slog.Logger
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	notificationRepo *repository.NotificationRepository,
	pipelineRepo *repository.PipelineRepository,
	emailPrefsRepo *repository.EmailPreferencesRepository,
	sentEmailsRepo *repository.SentEmailsRepository,
	promoRepo *repository.PromoCodeRepository,
	emailSvc *email.Service,
	sc *stripe.Client,
	promoCouponID string,
	appURL string,
	workosAPIKey string,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		pipelineRepo:     pipelineRepo,
		emailPrefsRepo:   emailPrefsRepo,
		sentEmailsRepo:   sentEmailsRepo,
		promoRepo:        promoRepo,
		emailSvc:         emailSvc,
		sc:               sc,
		promoCouponID:    promoCouponID,
		appURL:           appURL,
		workosAPIKey:     workosAPIKey,
		logger:           logger,
	}
}

type adminUserResponse struct {
	ID              int    `json:"id"`
	Email           string `json:"email"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Plan            string `json:"plan"`
	IsAdmin         bool   `json:"is_admin"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PendingExport   bool   `json:"pending_export"`
	PendingDeletion bool   `json:"pending_deletion"`
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
			ID:              u.ID,
			Email:           u.Email,
			FirstName:       u.FirstName,
			LastName:        u.LastName,
			Plan:            u.Plan,
			IsAdmin:         u.IsAdmin,
			CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			PendingExport:   u.PendingExport,
			PendingDeletion: u.PendingDeletion,
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

	notifications, total, err := h.notificationRepo.List(r.Context(), userID, false, page, 50)
	if err != nil {
		h.logger.Error("admin list notifications failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"limit":         50,
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

func (h *AdminHandler) GetPipelineRunDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	detail, err := h.pipelineRepo.GetPipelineRunDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineRunNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.logger.Error("get pipeline run detail failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
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

func (h *AdminHandler) ListMcpUsage(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	events, total, err := h.pipelineRepo.ListMcpUsage(r.Context(), page, 50)
	if err != nil {
		h.logger.Error("list mcp usage failed", "error", err)
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

type dqFilters struct {
	Page         int
	Limit        int
	Sort         string
	Order        string
	Resolved     *bool
	SnapshotDate string
	IssueType    string
	FieldName    string
}

func parseDQListParams(r *http.Request) dqFilters {
	f := dqFilters{Page: 1, Limit: 50}
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			f.Page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			f.Limit = v
		}
	}
	f.Sort = r.URL.Query().Get("sort")
	f.Order = r.URL.Query().Get("order")
	if rv := r.URL.Query().Get("resolved"); rv != "" {
		b := rv == "true"
		f.Resolved = &b
	}
	f.SnapshotDate = r.URL.Query().Get("snapshot_date")
	f.IssueType = r.URL.Query().Get("issue_type")
	f.FieldName = r.URL.Query().Get("field_name")
	return f
}

func (h *AdminHandler) ListDataQualityIssues(w http.ResponseWriter, r *http.Request) {
	f := parseDQListParams(r)

	items, total, err := h.pipelineRepo.ListDataQualityIssues(r.Context(), f.Page, f.Limit, f.Sort, f.Order, f.Resolved, f.SnapshotDate, f.IssueType, f.FieldName)
	if err != nil {
		h.logger.Error("list data quality issues failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
		"page":  f.Page,
		"limit": f.Limit,
	})
}

func (h *AdminHandler) DataQualitySummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.pipelineRepo.DataQualitySummary(r.Context())
	if err != nil {
		h.logger.Error("data quality summary failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []repository.DQSummaryRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *AdminHandler) GetDataQualityDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	detail, err := h.pipelineRepo.GetDataQualityDetail(r.Context(), id)
	if err != nil {
		h.logger.Error("get data quality detail failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *AdminHandler) UpdateDataQualityResolution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Resolved       bool   `json:"resolved"`
		ResolutionNote string `json:"resolution_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.pipelineRepo.UpdateDataQualityResolution(r.Context(), id, body.Resolved, body.ResolutionNote); err != nil {
		h.logger.Error("update data quality resolution failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListReconcileDQIssues(w http.ResponseWriter, r *http.Request) {
	f := parseDQListParams(r)

	items, total, err := h.pipelineRepo.ListReconcileDQIssues(r.Context(), f.Page, f.Limit, f.Sort, f.Order, f.Resolved, f.SnapshotDate, f.IssueType, f.FieldName)
	if err != nil {
		h.logger.Error("list reconcile dq issues failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
		"page":  f.Page,
		"limit": f.Limit,
	})
}

func (h *AdminHandler) ReconcileDQSummary(w http.ResponseWriter, r *http.Request) {
	items, err := h.pipelineRepo.ReconcileDQSummary(r.Context())
	if err != nil {
		h.logger.Error("reconcile dq summary failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []repository.DQSummaryRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *AdminHandler) GetReconcileDQDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	detail, err := h.pipelineRepo.GetReconcileDQDetail(r.Context(), id)
	if err != nil {
		h.logger.Error("get reconcile dq detail failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *AdminHandler) UpdateReconcileDQResolution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Resolved       bool   `json:"resolved"`
		ResolutionNote string `json:"resolution_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.pipelineRepo.UpdateReconcileDQResolution(r.Context(), id, body.Resolved, body.ResolutionNote); err != nil {
		h.logger.Error("update reconcile dq resolution failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListEmailPreferences(w http.ResponseWriter, r *http.Request) {
	prefs, err := h.emailPrefsRepo.ListAllWithUsers(r.Context())
	if err != nil {
		h.logger.Error("list email preferences failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

func (h *AdminHandler) UpdateEmailPreference(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	var body struct {
		SearchAlerts      bool `json:"search_alerts"`
		OpportunityAlerts bool `json:"opportunity_alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.emailPrefsRepo.Upsert(r.Context(), userID, body.SearchAlerts, body.OpportunityAlerts); err != nil {
		h.logger.Error("admin update email preference failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListSentEmails(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	emails, total, err := h.sentEmailsRepo.List(r.Context(), page, 50)
	if err != nil {
		h.logger.Error("list sent emails failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"emails": emails,
		"total":  total,
		"page":   page,
		"limit":  50,
	})
}

// coerceFloats converts float64 values (from JSON unmarshal) to int where
// possible, so Go templates can compare them with integer literals via eq.
func coerceFloats(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case float64:
			if val == float64(int(val)) {
				m[k] = int(val)
			}
		case map[string]any:
			coerceFloats(val)
		case []any:
			for _, item := range val {
				if sub, ok := item.(map[string]any); ok {
					coerceFloats(sub)
				}
			}
		}
	}
}

var allowedTemplates = map[string]bool{
	"digest.html":        true,
	"promo-invite.html":  true,
	"promo-revoked.html": true,
}

func (h *AdminHandler) SendNewEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TemplateName string         `json:"template_name"`
		ToEmail      string         `json:"to_email"`
		Subject      string         `json:"subject"`
		TemplateData map[string]any `json:"template_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if body.TemplateName == "" || body.ToEmail == "" || body.Subject == "" {
		http.Error(w, "template_name, to_email, and subject are required", http.StatusBadRequest)
		return
	}

	if !allowedTemplates[body.TemplateName] {
		http.Error(w, "template_name not allowed", http.StatusBadRequest)
		return
	}

	if h.emailSvc == nil {
		http.Error(w, "email service not configured", http.StatusServiceUnavailable)
		return
	}

	coerceFloats(body.TemplateData)

	sentID, err := h.emailSvc.SendEmail(r.Context(), email.SendEmailInput{
		UserID:       nil,
		ToEmail:      body.ToEmail,
		EmailType:    "admin_test",
		TemplateName: body.TemplateName,
		TemplateData: body.TemplateData,
		Subject:      body.Subject,
	})
	if err != nil {
		h.logger.Error("send new email failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": sentID})
}

func (h *AdminHandler) ResendEmail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var body struct {
		ToEmail string `json:"to_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ToEmail == "" {
		http.Error(w, "to_email is required", http.StatusBadRequest)
		return
	}

	if h.emailSvc == nil {
		http.Error(w, "email service not configured", http.StatusServiceUnavailable)
		return
	}

	if err := h.emailSvc.ResendExistingEmail(r.Context(), id, body.ToEmail); err != nil {
		h.logger.Error("resend email failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ExportUserData(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("export user data: get user failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	export, err := h.userRepo.ExportUserData(r.Context(), userID, user.Email)
	if err != nil {
		h.logger.Error("export user data failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(export)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("delete user: get user failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Prevent self-deletion
	requestingWorkOSID := authmw.UserIDFromContext(r.Context())
	if requestingWorkOSID == user.WorkOSID {
		http.Error(w, "cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Delete from WorkOS (revoke sessions + delete user)
	if h.workosAPIKey != "" && user.WorkOSID != "" {
		if err := h.deleteWorkOSUser(user.WorkOSID); err != nil {
			h.logger.Error("workos user deletion failed", "workos_id", user.WorkOSID, "error", err)
			// Continue with DB deletion even if WorkOS fails
		}
	}

	if err := h.userRepo.DeleteUser(r.Context(), userID, user.Email); err != nil {
		h.logger.Error("delete user failed", "user_id", userID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("user deleted", "user_id", userID, "email", user.Email)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	if h.sc == nil || h.promoCouponID == "" {
		http.Error(w, "promo codes not configured", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		UserID        int `json:"user_id"`
		ExpiresInDays int `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), body.UserID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if user.Plan == "pro" {
		http.Error(w, "user is already pro", http.StatusConflict)
		return
	}

	// Ensure user has a Stripe customer
	customerID := ""
	if user.StripeCustomerID != nil {
		customerID = *user.StripeCustomerID
	} else {
		cust, custErr := h.sc.V1Customers.Create(r.Context(), &stripe.CustomerCreateParams{
			Email: stripe.String(user.Email),
			Name:  stripe.String(user.FirstName + " " + user.LastName),
			Metadata: map[string]string{
				"govtrove_user_id": fmt.Sprintf("%d", user.ID),
				"workos_id":        user.WorkOSID,
			},
		})
		if custErr != nil {
			h.logger.Error("stripe customer create failed", "error", custErr)
			http.Error(w, "failed to create stripe customer", http.StatusInternalServerError)
			return
		}
		customerID = cust.ID
		_ = h.userRepo.SetStripeCustomerID(r.Context(), user.ID, customerID)
	}

	// Generate a readable code (Stripe only allows [a-zA-Z0-9\-_])
	randBytes := make([]byte, 2)
	rand.Read(randBytes)
	suffix := strings.ToUpper(hex.EncodeToString(randBytes))
	asciiOnly := regexp.MustCompile(`[^a-zA-Z0-9]`)
	firstName := strings.ToUpper(asciiOnly.ReplaceAllString(user.FirstName, ""))
	if firstName == "" {
		firstName = "USER"
	}
	code := fmt.Sprintf("GOVTROVE-%s-%s", firstName, suffix)

	promoParams := &stripe.PromotionCodeCreateParams{
		Coupon:         stripe.String(h.promoCouponID),
		Code:           stripe.String(code),
		MaxRedemptions: stripe.Int64(1),
		Customer:       stripe.String(customerID),
	}

	var expiresAt *time.Time
	if body.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(body.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
		promoParams.ExpiresAt = stripe.Int64(t.Unix())
	}

	stripePromo, err := h.sc.V1PromotionCodes.Create(r.Context(), promoParams)
	if err != nil {
		h.logger.Error("stripe promotion code create failed", "error", err)
		http.Error(w, "failed to create promotion code", http.StatusInternalServerError)
		return
	}

	userID := user.ID
	id, err := h.promoRepo.Create(r.Context(), stripePromo.Code, stripePromo.ID, &userID, expiresAt)
	if err != nil {
		h.logger.Error("failed to save promo code", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	inviteURL := fmt.Sprintf("%s/profile?promo=%s", h.appURL, stripePromo.Code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":         id,
		"code":       stripePromo.Code,
		"invite_url": inviteURL,
	})
}

func (h *AdminHandler) ListPromoCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := h.promoRepo.List(r.Context())
	if err != nil {
		h.logger.Error("list promo codes failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if codes == nil {
		codes = []repository.PromoCodeRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codes)
}

func (h *AdminHandler) SendPromoInvite(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if h.emailSvc == nil {
		http.Error(w, "email service not configured", http.StatusServiceUnavailable)
		return
	}

	codes, err := h.promoRepo.List(r.Context())
	if err != nil {
		h.logger.Error("list promo codes for send failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var promo *repository.PromoCodeRow
	for i := range codes {
		if codes[i].ID == id {
			promo = &codes[i]
			break
		}
	}
	if promo == nil {
		http.Error(w, "promo code not found", http.StatusNotFound)
		return
	}
	if promo.ForUserEmail == nil {
		http.Error(w, "no user associated with this promo code", http.StatusBadRequest)
		return
	}

	firstName := "there"
	if promo.ForUserName != nil && *promo.ForUserName != "" && *promo.ForUserName != " " {
		firstName = strings.SplitN(*promo.ForUserName, " ", 2)[0]
	}

	inviteURL := fmt.Sprintf("%s/profile?promo=%s", h.appURL, promo.Code)
	userID := promo.ForUserID

	_, err = h.emailSvc.SendEmail(r.Context(), email.SendEmailInput{
		UserID:       userID,
		ToEmail:      *promo.ForUserEmail,
		EmailType:    "promo_invite",
		TemplateName: "promo-invite.html",
		Subject:      "You're invited to GovTrove Pro",
		TemplateData: map[string]any{
			"FirstName": firstName,
			"InviteURL": inviteURL,
			"Code":      promo.Code,
		},
	})
	if err != nil {
		h.logger.Error("send promo invite failed", "error", err)
		http.Error(w, "failed to send invite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) RevokePromoCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if h.sc == nil {
		http.Error(w, "stripe not configured", http.StatusServiceUnavailable)
		return
	}

	promo, err := h.promoRepo.GetByID(r.Context(), id)
	if err != nil || promo == nil {
		http.Error(w, "promo code not found", http.StatusNotFound)
		return
	}
	if promo.RevokedAt != nil {
		http.Error(w, "promo code already revoked", http.StatusConflict)
		return
	}

	// Deactivate in Stripe
	_, err = h.sc.V1PromotionCodes.Update(r.Context(), promo.StripePromoID, &stripe.PromotionCodeUpdateParams{
		Active: stripe.Bool(false),
	})
	if err != nil {
		h.logger.Error("stripe promo deactivate failed", "error", err)
		http.Error(w, "failed to deactivate in Stripe", http.StatusInternalServerError)
		return
	}

	// If redeemed, cancel the subscription and downgrade the user
	if promo.RedeemedBy != nil {
		user, userErr := h.userRepo.GetByID(r.Context(), *promo.RedeemedBy)
		if userErr == nil && user != nil && user.SubscriptionID != nil {
			_, cancelErr := h.sc.V1Subscriptions.Cancel(r.Context(), *user.SubscriptionID, nil)
			if cancelErr != nil {
				h.logger.Error("failed to cancel subscription for revoked promo", "user_id", user.ID, "error", cancelErr)
			}
		}
		status := "revoked"
		if userErr == nil && user != nil {
			_ = h.userRepo.UpdateSubscription(r.Context(), user.ID, "free", nil, &status, false, nil)
			h.logger.Info("user downgraded due to promo revocation", "user_id", user.ID)

			if h.emailSvc != nil {
				firstName := user.FirstName
				if firstName == "" {
					firstName = "there"
				}
				_, emailErr := h.emailSvc.SendEmail(r.Context(), email.SendEmailInput{
					UserID:       &user.ID,
					ToEmail:      user.Email,
					EmailType:    "promo_revoked",
					TemplateName: "promo-revoked.html",
					Subject:      "Your GovTrove Pro access has ended",
					TemplateData: map[string]any{
						"FirstName":  firstName,
						"ProfileURL": h.appURL + "/profile",
					},
				})
				if emailErr != nil {
					h.logger.Error("failed to send promo revoked email", "user_id", user.ID, "error", emailErr)
				}
			}
		}
	}

	if err := h.promoRepo.Revoke(r.Context(), id); err != nil {
		h.logger.Error("failed to revoke promo code", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("promo code revoked", "id", id, "code", promo.Code)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) GetSnapCSVRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	record, err := h.pipelineRepo.GetSnapCSVRecord(r.Context(), id)
	if err != nil {
		h.logger.Error("get snap csv record failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if record == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *AdminHandler) GetSnapArchivedCSVRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	record, err := h.pipelineRepo.GetSnapArchivedCSVRecord(r.Context(), id)
	if err != nil {
		h.logger.Error("get snap archived csv record failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if record == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *AdminHandler) GetSnapAPIRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	record, err := h.pipelineRepo.GetSnapAPIRecord(r.Context(), id)
	if err != nil {
		h.logger.Error("get snap api record failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if record == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *AdminHandler) deleteWorkOSUser(workosID string) error {
	client := &http.Client{}

	// Delete user (which also invalidates all sessions)
	req, err := http.NewRequest("DELETE",
		fmt.Sprintf("https://api.workos.com/user_management/users/%s", workosID), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.workosAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deleting workos user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("workos delete returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

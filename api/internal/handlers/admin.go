package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/handriss/govtrove/api/internal/email"
	"github.com/handriss/govtrove/api/internal/repository"
)

type AdminHandler struct {
	userRepo       *repository.UserRepository
	userUpdateRepo *repository.UserUpdateRepository
	pipelineRepo   *repository.PipelineRepository
	emailPrefsRepo *repository.EmailPreferencesRepository
	sentEmailsRepo *repository.SentEmailsRepository
	emailSvc       *email.Service
	logger         *slog.Logger
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	userUpdateRepo *repository.UserUpdateRepository,
	pipelineRepo *repository.PipelineRepository,
	emailPrefsRepo *repository.EmailPreferencesRepository,
	sentEmailsRepo *repository.SentEmailsRepository,
	emailSvc *email.Service,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		userUpdateRepo: userUpdateRepo,
		pipelineRepo:   pipelineRepo,
		emailPrefsRepo: emailPrefsRepo,
		sentEmailsRepo: sentEmailsRepo,
		emailSvc:       emailSvc,
		logger:         logger,
	}
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

func parseDQListParams(r *http.Request) (page, limit int, sort, order string, resolved *bool) {
	page = 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	limit = 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	sort = r.URL.Query().Get("sort")
	order = r.URL.Query().Get("order")
	if rv := r.URL.Query().Get("resolved"); rv != "" {
		b := rv == "true"
		resolved = &b
	}
	return
}

func (h *AdminHandler) ListDataQualityIssues(w http.ResponseWriter, r *http.Request) {
	page, limit, sort, order, resolved := parseDQListParams(r)

	items, total, err := h.pipelineRepo.ListDataQualityIssues(r.Context(), page, limit, sort, order, resolved)
	if err != nil {
		h.logger.Error("list data quality issues failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
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
	page, limit, sort, order, resolved := parseDQListParams(r)

	items, total, err := h.pipelineRepo.ListReconcileDQIssues(r.Context(), page, limit, sort, order, resolved)
	if err != nil {
		h.logger.Error("list reconcile dq issues failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
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
	"welcome.html":            true,
	"opportunity_update.html": true,
	"search_results.html":     true,
	"digest.html":             true,
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
		UserID:       0,
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

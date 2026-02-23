package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/ogimage"
	"github.com/handriss/govtrove/api/internal/repository"
)

type OpportunityHandler struct {
	repo     *repository.OpportunityRepository
	logger   *slog.Logger
	renderer *ogimage.Renderer
}

func NewOpportunityHandler(repo *repository.OpportunityRepository, renderer *ogimage.Renderer, logger *slog.Logger) *OpportunityHandler {
	return &OpportunityHandler{repo: repo, logger: logger, renderer: renderer}
}

func (h *OpportunityHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)

	result, err := h.repo.Search(r.Context(), params)
	if err != nil {
		h.logger.Error("search failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

func (h *OpportunityHandler) GetFacets(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)
	result, err := h.repo.GetFacetCounts(r.Context(), params)
	if err != nil {
		h.logger.Error("get facets failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	h.writeJSON(w, http.StatusOK, result)
}

func (h *OpportunityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	opp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("get opportunity failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if opp == nil {
		http.Error(w, "Opportunity not found", http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, opp)
}

func (h *OpportunityHandler) GetSolicitationHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	history, err := h.repo.GetSolicitationHistory(r.Context(), id)
	if err != nil {
		h.logger.Error("get solicitation history failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if history == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{
			"solicitation_number": "",
			"total_notices":      0,
			"notices":            []any{},
			"truncated":          false,
		})
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	h.writeJSON(w, http.StatusOK, history)
}

func (h *OpportunityHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	options, err := h.repo.GetFilterOptions(r.Context())
	if err != nil {
		h.logger.Error("get filters failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, options)
}

var ogTemplate = template.Must(template.New("og").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta property="og:type" content="website">
<meta property="og:site_name" content="GovTrove">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:url" content="{{.URL}}">
<meta property="og:image" content="{{.ImageURL}}">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="{{.Title}}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Description}}">
<meta name="twitter:image" content="{{.ImageURL}}">{{if .Deadline}}
<meta name="twitter:label1" content="Deadline">
<meta name="twitter:data1" content="{{.Deadline}}">{{end}}{{if .SetAside}}
<meta name="twitter:label2" content="Set-Aside">
<meta name="twitter:data2" content="{{.SetAside}}">{{end}}
<title>{{.Title}} — GovTrove</title>
</head>
<body><p><a href="{{.URL}}">View on GovTrove</a></p></body>
</html>`))

type ogData struct {
	Title       string
	Description string
	URL         string
	ImageURL    string
	Deadline    string
	SetAside    string
}

func (h *OpportunityHandler) GetOGCard(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.renderOGFallback(w, idStr)
		return
	}

	opp, err := h.repo.GetByID(r.Context(), id)
	if err != nil || opp == nil {
		h.renderOGFallback(w, idStr)
		return
	}

	agency := ""
	if opp.Department != nil {
		agency = ogimage.ShortenAgency(*opp.Department)
	}

	title := opp.Title
	if agency != "" {
		combined := title + " — " + agency
		if len(combined) <= 60 {
			title = combined
		} else if len(title) > 57 {
			title = title[:57] + "..."
		}
	}

	var descParts []string
	if opp.SetAsideDesc != nil {
		descParts = append(descParts, *opp.SetAsideDesc)
	}
	if opp.Type != nil {
		descParts = append(descParts, *opp.Type)
	}
	if opp.ResponseDeadline != nil {
		descParts = append(descParts, "Deadline: "+opp.ResponseDeadline.Format("Jan 2, 2006"))
	}
	if opp.NAICSCode != nil {
		descParts = append(descParts, "NAICS: "+*opp.NAICSCode)
	}

	desc := "Federal Contract Opportunity"
	if len(descParts) > 0 {
		desc = strings.Join(descParts, " | ")
	}

	deadline := ""
	if opp.ResponseDeadline != nil {
		deadline = opp.ResponseDeadline.Format("Jan 2, 2006")
	}

	setAside := ""
	if opp.SetAsideDesc != nil {
		setAside = *opp.SetAsideDesc
	}

	data := ogData{
		Title:       title,
		Description: desc,
		URL:         fmt.Sprintf("https://app.govtrove.com/opportunity/%d", opp.ID),
		ImageURL:    fmt.Sprintf("https://api.govtrove.com/og/opportunities/%d/card.png", opp.ID),
		Deadline:    deadline,
		SetAside:    setAside,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := ogTemplate.Execute(w, data); err != nil {
		h.logger.Error("failed to render OG template", "error", err)
	}
}

func (h *OpportunityHandler) GetOGImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	opp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("get opportunity for OG image failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if opp == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data := ogimage.OpportunityData{
		ID:    opp.ID,
		Title: opp.Title,
	}
	if opp.Type != nil {
		data.Type = *opp.Type
	}
	if opp.Department != nil {
		data.Department = *opp.Department
	}
	if opp.SetAsideDesc != nil {
		data.SetAsideDesc = *opp.SetAsideDesc
	}
	if opp.SetAsideCode != nil {
		data.SetAsideCode = *opp.SetAsideCode
	}
	if opp.NAICSCode != nil {
		data.NAICSCode = *opp.NAICSCode
	}
	data.ResponseDeadline = opp.ResponseDeadline

	png, err := h.renderer.Render(data)
	if err != nil {
		h.logger.Error("render OG image failed", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(png)
}

func (h *OpportunityHandler) renderOGFallback(w http.ResponseWriter, idStr string) {
	data := ogData{
		Title:       "Federal Contract Opportunity",
		Description: "Search and discover federal contract opportunities on GovTrove",
		URL:         fmt.Sprintf("https://app.govtrove.com/opportunity/%s", idStr),
		ImageURL:    "https://govtrove.com/og-image.png",
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := ogTemplate.Execute(w, data); err != nil {
		h.logger.Error("failed to render OG fallback template", "error", err)
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func capSlice(s []string, max int) []string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func (h *OpportunityHandler) parseSearchParams(r *http.Request) models.SearchParams {
	q := r.URL.Query()

	params := models.SearchParams{
		Query:  truncate(q.Get("q"), 500),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
		Page:   1,
		Limit:  25,
	}

	if typeStr := q.Get("type"); typeStr != "" {
		params.Types = capSlice(strings.Split(typeStr, ","), 50)
	}

	if setAsideStr := q.Get("set_aside"); setAsideStr != "" {
		params.SetAsides = capSlice(strings.Split(setAsideStr, ","), 50)
	}

	if naicsStr := q.Get("naics"); naicsStr != "" {
		params.NAICSCodes = capSlice(strings.Split(naicsStr, ","), 50)
	}

	params.NAICSPrefix = truncate(q.Get("naics_prefix"), 10)

	if pscStr := q.Get("psc"); pscStr != "" {
		params.PSCCodes = capSlice(strings.Split(pscStr, ","), 50)
	}
	params.PSCPrefix = truncate(q.Get("psc_prefix"), 10)

	params.Department = truncate(q.Get("department"), 200)

	if stateStr := q.Get("state"); stateStr != "" {
		params.States = capSlice(strings.Split(stateStr, ","), 50)
	}

	if postedFrom := q.Get("posted_from"); postedFrom != "" {
		if t, err := time.Parse("2006-01-02", postedFrom); err == nil {
			params.PostedFrom = &t
		}
	}

	if postedTo := q.Get("posted_to"); postedTo != "" {
		if t, err := time.Parse("2006-01-02", postedTo); err == nil {
			params.PostedTo = &t
		}
	}

	if deadlineFrom := q.Get("deadline_from"); deadlineFrom != "" {
		if t, err := time.Parse("2006-01-02", deadlineFrom); err == nil {
			params.DeadlineFrom = &t
		}
	}

	if deadlineTo := q.Get("deadline_to"); deadlineTo != "" {
		if t, err := time.Parse("2006-01-02", deadlineTo); err == nil {
			params.DeadlineTo = &t
		}
	}

	if pageStr := q.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if params.Page > 10000 {
		params.Page = 10000
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}

	return params
}

func (h *OpportunityHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

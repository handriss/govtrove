package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/ogimage"
	"github.com/handriss/govtrove/api/internal/repository"
	"github.com/handriss/govtrove/api/internal/searchrescue"
)

// Below this many hits a quoted search is worth re-probing unquoted; above it
// the user has plenty to work with and the extra query isn't worth the latency.
const quoteRelaxThreshold = 10

type OpportunityHandler struct {
	repo     *repository.OpportunityRepository
	logger   *slog.Logger
	renderer *ogimage.Renderer
	eventLog *EventLogger
	userRepo *repository.UserRepository
	geoRepo  *repository.GeoSynonymRepository
	rescue   *searchrescue.Service // nil unless SEARCH_RESCUE_ENABLED
}

// SetRescueService enables the zero-result search rescue endpoint.
func (h *OpportunityHandler) SetRescueService(svc *searchrescue.Service) {
	h.rescue = svc
}

func NewOpportunityHandler(repo *repository.OpportunityRepository, renderer *ogimage.Renderer, logger *slog.Logger, eventLog *EventLogger, userRepo *repository.UserRepository, geoRepo *repository.GeoSynonymRepository) *OpportunityHandler {
	return &OpportunityHandler{repo: repo, logger: logger, renderer: renderer, eventLog: eventLog, userRepo: userRepo, geoRepo: geoRepo}
}

func (h *OpportunityHandler) resolveOptionalUserID(r *http.Request) *int {
	workosID := authmw.UserIDFromContext(r.Context())
	if workosID == "" {
		return nil
	}
	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		return nil
	}
	return &user.ID
}

func (h *OpportunityHandler) expandGeoSynonyms(r *http.Request, params *models.SearchParams) {
	if params.Query == "" || h.geoRepo == nil {
		return
	}
	matches, err := h.geoRepo.Lookup(r.Context(), params.Query)
	if err != nil {
		h.logger.Warn("geo synonym lookup failed", "error", err)
		return
	}
	for _, m := range matches {
		params.GeoStates = append(params.GeoStates, m.States...)
		params.GeoCities = append(params.GeoCities, m.Cities...)
	}
}

func (h *OpportunityHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)
	h.expandGeoSynonyms(r, &params)

	start := time.Now()
	result, err := h.repo.Search(r.Context(), params)
	durationMs := int(time.Since(start).Milliseconds())
	if err != nil {
		// 57014 = query_canceled, i.e. the statement timeout fired. Almost always a
		// quoted phrase whose only real word is very common. Say so, so the user can
		// act, instead of returning a bare 500 they can do nothing with.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "57014" {
			h.logger.Warn("search timed out", "query", params.Query)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "That search was too broad to finish. Try adding a word, or narrowing with a NAICS or deadline filter.",
			})
			return
		}
		h.logger.Error("search failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if result.Total == 0 && params.Query != "" {
		suggestQuery := strings.TrimSpace(strings.ReplaceAll(params.Query, `"`, ""))
		if suggestQuery != "" {
			if suggestion, err := h.repo.SuggestQuery(r.Context(), suggestQuery); err == nil && suggestion != "" {
				result.Suggestion = suggestion
			} else if err != nil {
				h.logger.Warn("suggest query failed", "error", err)
			}
		}
	}

	// Quotes are an exact-substring match, so they can cut thousands of hits to
	// one with no visible reason. Only worth a second query when the quoted
	// result set is small enough that the user is probably stuck.
	if result.Total < quoteRelaxThreshold && strings.Contains(params.Query, `"`) {
		relaxed := strings.TrimSpace(strings.ReplaceAll(params.Query, `"`, ""))
		if relaxed != "" && relaxed != params.Query {
			probe := params
			probe.Query = relaxed
			probe.Page, probe.Limit = 1, 1
			if r2, err := h.repo.Search(r.Context(), probe); err == nil && r2.Total > result.Total {
				result.RelaxedQuery = relaxed
				result.RelaxedTotal = r2.Total
			} else if err != nil {
				h.logger.Warn("relaxed-query probe failed", "error", err)
			}
		}
	}

	h.writeJSON(w, http.StatusOK, result)

	userID := h.resolveOptionalUserID(r)
	event := &models.SearchEvent{EventType: "search", DurationMs: &durationMs}
	if params.Query != "" {
		event.Query = &params.Query
	}
	if params.Sort != "" {
		event.SortBy = &params.Sort
	}
	if params.Page > 0 {
		event.Page = &params.Page
	}
	event.TotalResults = &result.Total
	event.Filters = h.buildFilters(params)
	h.eventLog.Log(r, userID, event)
}

func (h *OpportunityHandler) GetFacets(w http.ResponseWriter, r *http.Request) {
	params := h.parseSearchParams(r)
	h.expandGeoSynonyms(r, &params)
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

	userID := h.resolveOptionalUserID(r)
	h.eventLog.Log(r, userID, &models.SearchEvent{
		EventType:     "view",
		OpportunityID: &id,
	})
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
		Query:      truncate(q.Get("q"), 500),
		Sort:       q.Get("sort"),
		Order:      q.Get("order"),
		Page:       1,
		Limit:      25,
	}

	if typeStr := q.Get("type"); typeStr != "" {
		raw := capSlice(strings.Split(typeStr, ","), 50)
		params.Types = models.NormalizeNoticeTypes(raw)
	}

	if setAsideStr := q.Get("set_aside"); setAsideStr != "" {
		params.SetAsides = capSlice(strings.Split(setAsideStr, ","), 50)
	}

	if naicsStr := q.Get("naics"); naicsStr != "" {
		params.NAICSCodes = capSlice(strings.Split(naicsStr, ","), 50)
	}

	params.NAICSPrefix = truncate(q.Get("naics_prefix"), 10)

	if np := q.Get("naics_prefixes"); np != "" {
		params.NAICSPrefixes = capSlice(strings.Split(np, ","), 10)
	}

	if pscStr := q.Get("psc"); pscStr != "" {
		params.PSCCodes = capSlice(strings.Split(pscStr, ","), 50)
	}
	params.PSCPrefix = truncate(q.Get("psc_prefix"), 10)

	if pp := q.Get("psc_prefixes"); pp != "" {
		params.PSCPrefixes = capSlice(strings.Split(pp, ","), 10)
	}

	params.Department = truncate(q.Get("department"), 200)

	if agencyStr := q.Get("agency"); agencyStr != "" {
		params.AgencyPaths = capSlice(strings.Split(agencyStr, ","), 20)
	}

	if stateStr := q.Get("state"); stateStr != "" {
		params.States = capSlice(strings.Split(stateStr, ","), 50)
	}

	params.SolicitationNumber = truncate(q.Get("sol_num"), 100)
	params.PopCity = truncate(q.Get("pop_city"), 100)

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

	params.ActiveOnly = q.Get("active") == "true"

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

func (h *OpportunityHandler) buildFilters(params models.SearchParams) map[string]interface{} {
	f := make(map[string]interface{})
	if len(params.Types) > 0 {
		f["type"] = params.Types
	}
	if len(params.SetAsides) > 0 {
		f["set_aside"] = params.SetAsides
	}
	if len(params.NAICSCodes) > 0 {
		f["naics"] = params.NAICSCodes
	}
	if len(params.PSCCodes) > 0 {
		f["psc"] = params.PSCCodes
	}
	if len(params.States) > 0 {
		f["state"] = params.States
	}
	if params.Department != "" {
		f["department"] = params.Department
	}
	if len(params.AgencyPaths) > 0 {
		f["agency"] = params.AgencyPaths
	}
	if params.SolicitationNumber != "" {
		f["sol_num"] = params.SolicitationNumber
	}
	if params.PopCity != "" {
		f["pop_city"] = params.PopCity
	}
	if params.PostedFrom != nil {
		f["posted_from"] = params.PostedFrom.Format("2006-01-02")
	}
	if params.PostedTo != nil {
		f["posted_to"] = params.PostedTo.Format("2006-01-02")
	}
	if params.DeadlineFrom != nil {
		f["deadline_from"] = params.DeadlineFrom.Format("2006-01-02")
	}
	if params.DeadlineTo != nil {
		f["deadline_to"] = params.DeadlineTo.Format("2006-01-02")
	}
	return f
}

func (h *OpportunityHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

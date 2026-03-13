package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed templates/digest.html
var digestTemplateFS embed.FS

type EmailSender struct {
	apiKey            string
	from              string
	baseURL           string
	unsubscribeSecret string
	pool              *pgxpool.Pool
	logger            *slog.Logger
	tmpl              *template.Template
}

func NewEmailSender(apiKey, from, baseURL, unsubSecret string, pool *pgxpool.Pool, logger *slog.Logger) (*EmailSender, error) {
	tmpl, err := template.ParseFS(digestTemplateFS, "templates/digest.html")
	if err != nil {
		return nil, fmt.Errorf("parse digest template: %w", err)
	}
	return &EmailSender{
		apiKey:            apiKey,
		from:              from,
		baseURL:           baseURL,
		unsubscribeSecret: unsubSecret,
		pool:              pool,
		logger:            logger,
		tmpl:              tmpl,
	}, nil
}

type searchAlertData struct {
	MatchCount       int
	TotalResultCount int
	SearchName       string
	SearchURL        string
	TopOpportunities []oppLink
	HasMore          bool
	RemainingCount   int
}

type oppLink struct {
	URL   string
	Title string
}

type oppAlertData struct {
	OpportunityTitle   string
	SolicitationNumber string
	Summary            string
	OpportunityURL     string
}

type digestData struct {
	Greeting          string
	SearchAlerts      []searchAlertData
	OpportunityAlerts []oppAlertData
	UnsubscribeURL    string
}

func (e *EmailSender) SendDigest(ctx context.Context, userID int, toEmail, firstName string, searchNotifs, oppNotifs []notificationRow) error {
	greeting := "Hi there,"
	if firstName != "" {
		greeting = fmt.Sprintf("Hi %s,", firstName)
	}

	data := digestData{
		Greeting:       greeting,
		UnsubscribeURL: e.generateUnsubscribeURL(userID),
	}

	for _, n := range searchNotifs {
		sa, err := e.buildSearchAlert(ctx, n)
		if err != nil {
			e.logger.Warn("skip search notification", "id", n.ID, "error", err)
			continue
		}
		data.SearchAlerts = append(data.SearchAlerts, sa)
	}

	sort.Slice(data.SearchAlerts, func(i, j int) bool {
		return data.SearchAlerts[i].MatchCount < data.SearchAlerts[j].MatchCount
	})

	for _, n := range oppNotifs {
		oa, err := e.buildOppAlert(n)
		if err != nil {
			e.logger.Warn("skip opp notification", "id", n.ID, "error", err)
			continue
		}
		data.OpportunityAlerts = append(data.OpportunityAlerts, oa)
	}

	if len(data.SearchAlerts) == 0 && len(data.OpportunityAlerts) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := e.tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render digest: %w", err)
	}

	subject := e.buildSubject(data.SearchAlerts, data.OpportunityAlerts)

	resendID, err := e.callResendAPI(ctx, toEmail, subject, buf.String())
	if err != nil {
		return fmt.Errorf("send digest: %w", err)
	}

	e.logSentEmail(ctx, userID, toEmail, subject, resendID)
	return nil
}

func (e *EmailSender) buildSearchAlert(ctx context.Context, n notificationRow) (searchAlertData, error) {
	var details struct {
		SearchName    string          `json:"search_name"`
		SearchID      int             `json:"search_id"`
		SearchFilters json.RawMessage `json:"search_filters"`
		MatchCount    int             `json:"match_count"`
		Matches       []struct {
			ID    int     `json:"id"`
			Title *string `json:"title"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(n.Details), &details); err != nil {
		return searchAlertData{}, fmt.Errorf("unmarshal search details: %w", err)
	}

	searchURL := e.baseURL + "/"
	if len(details.SearchFilters) > 0 && string(details.SearchFilters) != "null" {
		var filters map[string]any
		if err := json.Unmarshal(details.SearchFilters, &filters); err == nil {
			searchURL = e.buildSearchURL(filters)
		}
	}

	matchCount := details.MatchCount
	if matchCount == 0 {
		matchCount = len(details.Matches)
	}

	sa := searchAlertData{
		MatchCount: matchCount,
		SearchName: details.SearchName,
		SearchURL:  searchURL,
	}

	if details.SearchID > 0 && e.pool != nil {
		_ = e.pool.QueryRow(ctx, `SELECT COALESCE(total_result_count, 0) FROM saved_searches WHERE id = $1`, details.SearchID).Scan(&sa.TotalResultCount)
	}

	limit := 3
	if len(details.Matches) < limit {
		limit = len(details.Matches)
	}
	for _, m := range details.Matches[:limit] {
		title := "Untitled"
		if m.Title != nil {
			title = *m.Title
		}
		sa.TopOpportunities = append(sa.TopOpportunities, oppLink{
			URL:   fmt.Sprintf("%s/opportunity/%d", e.baseURL, m.ID),
			Title: title,
		})
	}

	sa.HasMore = details.MatchCount > len(sa.TopOpportunities)
	sa.RemainingCount = details.MatchCount - len(sa.TopOpportunities)

	return sa, nil
}

func (e *EmailSender) buildOppAlert(n notificationRow) (oppAlertData, error) {
	var details struct {
		OpportunityID      int               `json:"opportunity_id"`
		Title              *string           `json:"title"`
		SolicitationNumber *string           `json:"solicitation_number"`
		ChangeType         string            `json:"change_type"`
		Changes            []json.RawMessage `json:"changes"`
	}
	if err := json.Unmarshal([]byte(n.Details), &details); err != nil {
		return oppAlertData{}, fmt.Errorf("unmarshal opp details: %w", err)
	}

	title := "Untitled"
	if details.Title != nil {
		title = *details.Title
	}
	solNum := ""
	if details.SolicitationNumber != nil {
		solNum = *details.SolicitationNumber
	}

	summary := "Fields updated"
	if details.ChangeType == "amendment" {
		summary = "Amendment posted"
	} else if len(details.Changes) > 0 {
		var fields []string
		for _, c := range details.Changes {
			var change struct {
				Field string `json:"field"`
			}
			if json.Unmarshal(c, &change) == nil && change.Field != "" {
				fields = append(fields, friendlyFieldName(change.Field))
			}
		}
		if len(fields) > 0 {
			summary = "Fields updated: " + joinFields(fields)
		}
	}

	return oppAlertData{
		OpportunityTitle:   title,
		SolicitationNumber: solNum,
		Summary:            summary,
		OpportunityURL:     fmt.Sprintf("%s/opportunity/%d", e.baseURL, details.OpportunityID),
	}, nil
}

func joinFields(fields []string) string {
	if len(fields) <= 3 {
		result := ""
		for i, f := range fields {
			if i > 0 {
				result += ", "
			}
			result += f
		}
		return result
	}
	return fields[0] + ", " + fields[1] + fmt.Sprintf(", +%d more", len(fields)-2)
}

// keyMap translates saved search filter JSON keys (camelCase) to frontend URL params.
var keyMap = map[string]string{
	"keyword":            "q",
	"naics":              "naics",
	"psc":                "psc",
	"setAside":           "set_aside",
	"noticeType":         "type",
	"agency":             "agency",
	"department":         "agency",
	"state":              "state",
	"postedFrom":         "posted_from",
	"postedTo":           "posted_to",
	"deadlinePreset":     "deadline",
	"deadlineFrom":       "deadline_from",
	"deadlineTo":         "deadline_to",
	"solicitationNumber": "sol_num",
	"popCity":            "pop_city",
	"activeOnly":         "active",
}

func (e *EmailSender) buildSearchURL(filters map[string]any) string {
	q := url.Values{}
	for k, v := range filters {
		param, ok := keyMap[k]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case string:
			if val != "" {
				q.Set(param, val)
			}
		case bool:
			if !val {
				q.Set(param, "false")
			}
		case []any:
			var items []string
			for _, item := range val {
				if s, ok := item.(string); ok && s != "" {
					items = append(items, s)
				}
			}
			if len(items) > 0 {
				q.Set(param, strings.Join(items, ","))
			}
		}
	}
	if len(q) == 0 {
		return e.baseURL + "/"
	}
	return e.baseURL + "/?" + q.Encode()
}

func (e *EmailSender) buildSubject(searches []searchAlertData, opps []oppAlertData) string {
	switch {
	case len(searches) == 1 && len(opps) == 0:
		return fmt.Sprintf("%d new match%s for \"%s\"",
			searches[0].MatchCount, plural(searches[0].MatchCount), searches[0].SearchName)
	case len(searches) > 1 && len(opps) == 0:
		return fmt.Sprintf("New matches across %d saved searches", len(searches))
	case len(searches) == 0 && len(opps) == 1:
		return fmt.Sprintf("Update: %s", truncate(opps[0].OpportunityTitle, 60))
	case len(searches) == 0 && len(opps) > 1:
		return fmt.Sprintf("%d updates to your saved opportunities", len(opps))
	case len(searches) > 0 && len(opps) > 0:
		searchPart := fmt.Sprintf("New matches across %d search%s", len(searches), plural(len(searches)))
		if len(searches) == 1 {
			searchPart = fmt.Sprintf("%d new match%s for \"%s\"",
				searches[0].MatchCount, plural(searches[0].MatchCount), searches[0].SearchName)
		}
		return fmt.Sprintf("%s + %d opportunity update%s",
			searchPart, len(opps), pluralS(len(opps)))
	default:
		return "Your GovTrove update"
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type resendResponse struct {
	ID string `json:"id"`
}

func (e *EmailSender) callResendAPI(ctx context.Context, to, subject, html string) (string, error) {
	body, err := json.Marshal(resendRequest{
		From:    e.from,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var r resendResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return r.ID, nil
}

func (e *EmailSender) logSentEmail(ctx context.Context, userID int, toEmail, subject, resendID string) {
	_, err := e.pool.Exec(ctx,
		`INSERT INTO sent_emails (user_id, to_email, email_type, template_name, subject, resend_message_id, status)
		 VALUES ($1, $2, 'digest', 'digest.html', $3, $4, 'sent')`,
		userID, toEmail, subject, resendID,
	)
	if err != nil {
		e.logger.Error("failed to log sent email", "error", err, "user_id", userID)
	}
}

func (e *EmailSender) generateUnsubscribeURL(userID int) string {
	uid := strconv.Itoa(userID)
	mac := hmac.New(sha256.New, []byte(e.unsubscribeSecret))
	mac.Write([]byte("unsubscribe:" + uid))
	token := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("https://api.govtrove.com/api/unsubscribe?token=%s&uid=%s", token, uid)
}

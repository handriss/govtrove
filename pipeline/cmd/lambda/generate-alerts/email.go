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
	"strconv"
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
		sa, err := e.buildSearchAlert(n)
		if err != nil {
			e.logger.Warn("skip search notification", "id", n.ID, "error", err)
			continue
		}
		data.SearchAlerts = append(data.SearchAlerts, sa)
	}

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

	subject := "Your GovTrove Update"
	if len(data.SearchAlerts) > 0 && len(data.OpportunityAlerts) == 0 {
		subject = "New matches for your saved searches"
	} else if len(data.SearchAlerts) == 0 && len(data.OpportunityAlerts) > 0 {
		subject = "Updates to your saved opportunities"
	}

	resendID, err := e.callResendAPI(ctx, toEmail, subject, buf.String())
	if err != nil {
		return fmt.Errorf("send digest: %w", err)
	}

	e.logSentEmail(ctx, userID, toEmail, subject, resendID)
	return nil
}

func (e *EmailSender) buildSearchAlert(n notificationRow) (searchAlertData, error) {
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

	sa := searchAlertData{
		MatchCount: details.MatchCount,
		SearchName: details.SearchName,
		SearchURL:  searchURL,
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

func (e *EmailSender) buildSearchURL(filters map[string]any) string {
	var params []string
	for k, v := range filters {
		switch val := v.(type) {
		case string:
			if val != "" {
				params = append(params, k+"="+val)
			}
		case bool:
			if val {
				params = append(params, k+"=true")
			}
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok && s != "" {
					params = append(params, k+"="+s)
				}
			}
		}
	}
	if len(params) == 0 {
		return e.baseURL + "/"
	}
	result := e.baseURL + "/?"
	for i, p := range params {
		if i > 0 {
			result += "&"
		}
		result += p
	}
	return result
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

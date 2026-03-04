package email

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var base64Std = base64.StdEncoding

//go:embed templates/*.html
var templateFS embed.FS

type Service struct {
	apiKey        string
	from          string
	webhookSecret string
	pool          *pgxpool.Pool
	logger        *slog.Logger
	templates     *template.Template
}

func NewService(apiKey, from, webhookSecret string, pool *pgxpool.Pool, logger *slog.Logger) *Service {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	return &Service{
		apiKey:        apiKey,
		from:          from,
		webhookSecret: webhookSecret,
		pool:          pool,
		logger:        logger,
		templates:     tmpl,
	}
}

type SendEmailInput struct {
	UserID       int
	ToEmail      string
	EmailType    string
	TemplateName string
	TemplateData map[string]any
	Subject      string
}

type resendRequest struct {
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type resendResponse struct {
	ID string `json:"id"`
}

func (s *Service) SendEmail(ctx context.Context, input SendEmailInput) (string, error) {
	html, err := s.RenderTemplate(input.TemplateName, input.TemplateData)
	if err != nil {
		return "", fmt.Errorf("rendering template %s: %w", input.TemplateName, err)
	}

	resendID, err := s.callResendAPI(ctx, input.ToEmail, input.Subject, html)
	if err != nil {
		return "", fmt.Errorf("calling resend API: %w", err)
	}

	templateDataJSON, _ := json.Marshal(input.TemplateData)

	var sentID string
	err = s.pool.QueryRow(ctx,
		`INSERT INTO sent_emails (user_id, to_email, email_type, template_name, template_data, subject, resend_message_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'sent')
		 RETURNING id`,
		input.UserID, input.ToEmail, input.EmailType, input.TemplateName,
		string(templateDataJSON), input.Subject, resendID,
	).Scan(&sentID)
	if err != nil {
		s.logger.Error("failed to record sent email", "error", err, "resend_id", resendID)
	}

	s.logger.Info("email sent", "to", input.ToEmail, "type", input.EmailType, "resend_id", resendID)
	return sentID, nil
}

func (s *Service) SendWelcome(ctx context.Context, to, firstName string) error {
	greeting := "Hi there,"
	if firstName != "" {
		greeting = fmt.Sprintf("Hi %s,", firstName)
	}

	data := map[string]any{
		"Greeting": greeting,
	}

	html, err := s.RenderTemplate("welcome.html", data)
	if err != nil {
		return fmt.Errorf("rendering welcome template: %w", err)
	}

	_, err = s.callResendAPI(ctx, to, "You're in — start exploring federal opportunities", html)
	if err != nil {
		return fmt.Errorf("sending welcome email to %s: %w", to, err)
	}

	s.logger.Info("welcome email sent", "to", to)
	return nil
}

func (s *Service) RenderTemplate(name string, data map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}
	return buf.String(), nil
}

func (s *Service) callResendAPI(ctx context.Context, to, subject, html string) (string, error) {
	reqBody := resendRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var resendResp resendResponse
	if err := json.Unmarshal(respBody, &resendResp); err != nil {
		return "", fmt.Errorf("parsing resend response: %w", err)
	}

	return resendResp.ID, nil
}

func (s *Service) ResendExistingEmail(ctx context.Context, sentEmailID, toEmail string) error {
	var templateName string
	var templateDataStr *string
	var subject string
	var userID int

	err := s.pool.QueryRow(ctx,
		`SELECT user_id, template_name, template_data, subject FROM sent_emails WHERE id = $1`,
		sentEmailID,
	).Scan(&userID, &templateName, &templateDataStr, &subject)
	if err != nil {
		return fmt.Errorf("fetching sent email %s: %w", sentEmailID, err)
	}

	var templateData map[string]any
	if templateDataStr != nil {
		json.Unmarshal([]byte(*templateDataStr), &templateData)
	}

	html, err := s.RenderTemplate(templateName, templateData)
	if err != nil {
		return fmt.Errorf("rendering template %s: %w", templateName, err)
	}

	resendID, err := s.callResendAPI(ctx, toEmail, subject, html)
	if err != nil {
		return fmt.Errorf("resending email: %w", err)
	}

	s.logger.Info("email resent", "original_id", sentEmailID, "to", toEmail, "resend_id", resendID)
	return nil
}

func (s *Service) GenerateUnsubscribeURL(userID int) string {
	uid := strconv.Itoa(userID)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte("unsubscribe:" + uid))
	token := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("https://api.govtrove.com/api/unsubscribe?token=%s&uid=%s", token, uid)
}

func (s *Service) ValidateUnsubscribeToken(userID int, token string) bool {
	uid := strconv.Itoa(userID)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte("unsubscribe:" + uid))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(token), []byte(expected))
}

// VerifyWebhookSignature verifies Resend/Svix webhook signatures.
// Svix signs: "{svix-id}.{svix-timestamp}.{body}" using HMAC-SHA256
// with the base64-decoded secret (after stripping the "whsec_" prefix).
// The svix-signature header contains "v1,{base64-signature}".
func (s *Service) VerifyWebhookSignature(payload []byte, svixID, svixTimestamp, svixSignature string) bool {
	if s.webhookSecret == "" || svixID == "" || svixTimestamp == "" || svixSignature == "" {
		return false
	}

	// Strip "whsec_" prefix and decode base64 secret
	secretStr := strings.TrimPrefix(s.webhookSecret, "whsec_")
	secretBytes, err := base64Decode(secretStr)
	if err != nil {
		return false
	}

	// Construct signed content: "{svix-id}.{svix-timestamp}.{body}"
	signedContent := svixID + "." + svixTimestamp + "." + string(payload)

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expected := base64Encode(mac.Sum(nil))

	// svix-signature may contain multiple signatures like "v1,sig1 v1,sig2"
	for _, sig := range strings.Split(svixSignature, " ") {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) != 2 {
			continue
		}
		if hmac.Equal([]byte(parts[1]), []byte(expected)) {
			return true
		}
	}
	return false
}

func base64Decode(s string) ([]byte, error) {
	return base64Std.DecodeString(s)
}

func base64Encode(b []byte) string {
	return base64Std.EncodeToString(b)
}

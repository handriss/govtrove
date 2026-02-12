package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var validSubjects = map[string]bool{
	"General":     true,
	"Support":     true,
	"Feedback":    true,
	"Partnership": true,
}

type ContactHandler struct {
	repo        *repository.ContactRepository
	snsClient   *sns.Client
	snsTopicARN string
	logger      *slog.Logger
}

func NewContactHandler(repo *repository.ContactRepository, snsClient *sns.Client, snsTopicARN string, logger *slog.Logger) *ContactHandler {
	return &ContactHandler{
		repo:        repo,
		snsClient:   snsClient,
		snsTopicARN: snsTopicARN,
		logger:      logger,
	}
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Website != "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)

	if req.Name == "" || req.Email == "" || req.Subject == "" || req.Message == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if len(req.Name) > 200 {
		http.Error(w, "Name must be 200 characters or less", http.StatusBadRequest)
		return
	}

	if len(req.Email) > 320 || !emailRegex.MatchString(req.Email) {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	if !validSubjects[req.Subject] {
		http.Error(w, "Invalid subject", http.StatusBadRequest)
		return
	}

	if len(req.Message) > 5000 {
		http.Error(w, "Message must be 5000 characters or less", http.StatusBadRequest)
		return
	}

	msg := &models.ContactMessage{
		Name:      req.Name,
		Email:     req.Email,
		Subject:   req.Subject,
		Message:   req.Message,
		IPAddress: r.RemoteAddr,
	}

	if err := h.repo.Create(r.Context(), msg); err != nil {
		h.logger.Error("create contact message failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if h.snsClient != nil && h.snsTopicARN != "" {
		go h.sendNotification(msg)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ContactHandler) sendNotification(msg *models.ContactMessage) {
	preview := msg.Message
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}

	body := fmt.Sprintf("New contact form submission:\n\nName: %s\nEmail: %s\nSubject: %s\n\n%s",
		msg.Name, msg.Email, msg.Subject, preview)

	_, err := h.snsClient.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(h.snsTopicARN),
		Subject:  aws.String(fmt.Sprintf("GovTrove Contact: %s", msg.Subject)),
		Message:  aws.String(body),
	})
	if err != nil {
		h.logger.Error("failed to send contact notification", "error", err)
	}
}

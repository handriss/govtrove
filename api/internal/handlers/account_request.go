package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

var validRequestTypes = map[string]bool{
	"data_export":      true,
	"account_deletion": true,
}

type AccountRequestHandler struct {
	repo        *repository.AccountRequestRepository
	userRepo    *repository.UserRepository
	snsClient   *sns.Client
	snsTopicARN string
	logger      *slog.Logger
}

func NewAccountRequestHandler(
	repo *repository.AccountRequestRepository,
	userRepo *repository.UserRepository,
	snsClient *sns.Client,
	snsTopicARN string,
	logger *slog.Logger,
) *AccountRequestHandler {
	return &AccountRequestHandler{
		repo:        repo,
		userRepo:    userRepo,
		snsClient:   snsClient,
		snsTopicARN: snsTopicARN,
		logger:      logger,
	}
}

func (h *AccountRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	workosID := authmw.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil {
		h.logger.Error("failed to look up user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req models.CreateAccountRequestInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !validRequestTypes[req.RequestType] {
		http.Error(w, "Invalid request_type", http.StatusBadRequest)
		return
	}

	ar := &models.AccountRequest{
		UserID:      user.ID,
		RequestType: req.RequestType,
		Status:      "pending",
	}

	if err := h.repo.Create(r.Context(), ar); err != nil {
		h.logger.Error("create account request failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if h.snsClient != nil && h.snsTopicARN != "" {
		go h.sendNotification(ar, user.Email)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AccountRequestHandler) sendNotification(ar *models.AccountRequest, email string) {
	label := "Data Export"
	if ar.RequestType == "account_deletion" {
		label = "Account Deletion"
	}

	subject := fmt.Sprintf("GovTrove: %s Request", label)
	body := fmt.Sprintf("%s request received:\n\nEmail: %s\nUser ID: %d\nRequest ID: %d",
		label, email, ar.UserID, ar.ID)

	_, err := h.snsClient.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(h.snsTopicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(body),
	})
	if err != nil {
		h.logger.Error("failed to send account request notification", "error", err)
	}
}

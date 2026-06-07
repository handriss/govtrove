package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type DonationHandler struct {
	userRepo    *repository.UserRepository
	snsClient   *sns.Client
	snsTopicARN string
	logger      *slog.Logger
}

func NewDonationHandler(userRepo *repository.UserRepository, snsClient *sns.Client, snsTopicARN string, logger *slog.Logger) *DonationHandler {
	return &DonationHandler{
		userRepo:    userRepo,
		snsClient:   snsClient,
		snsTopicARN: snsTopicARN,
		logger:      logger,
	}
}

func (h *DonationHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source string `json:"source"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	source := strings.TrimSpace(body.Source)
	if source == "" {
		source = "unknown"
	}

	userInfo := "anonymous"
	if workosID := middleware.UserIDFromContext(r.Context()); workosID != "" {
		if user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID); err == nil && user != nil {
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name == "" {
				userInfo = user.Email
			} else {
				userInfo = fmt.Sprintf("%s <%s>", name, user.Email)
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)

	if h.snsClient != nil && h.snsTopicARN != "" {
		go h.publish(userInfo, source)
	}
}

func (h *DonationHandler) publish(userInfo, source string) {
	body := fmt.Sprintf("Donation link clicked.\n\nUser: %s\nSource: %s\nTime: %s",
		userInfo, source, time.Now().UTC().Format(time.RFC3339))

	_, err := h.snsClient.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(h.snsTopicARN),
		Subject:  aws.String("GovTrove: Donation Link Clicked"),
		Message:  aws.String(body),
	})
	if err != nil {
		h.logger.Error("failed to send donation click notification", "error", err)
	}
}

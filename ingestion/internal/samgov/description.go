package samgov

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/handriss/govtrove/ingestion/internal/database"
)

const (
	DescriptionAPIBaseURL = "https://api.sam.gov/prod/opportunities/v1/noticedesc"
	DescriptionDelay      = 333 * time.Millisecond
)

type DescriptionClient struct {
	httpClient *TrackedHTTPClient
	logger     *slog.Logger
	apiKey     string
	baseURL    string
}

type DescriptionResponse struct {
	Description string `json:"description"`
}

type DescriptionResult struct {
	Fetched int
	Errors  int
}

func NewDescriptionClient(db *database.DB, logger *slog.Logger, apiKey string) *DescriptionClient {
	return &DescriptionClient{
		httpClient: NewTrackedHTTPClient(db, logger),
		logger:     logger,
		apiKey:     apiKey,
		baseURL:    DescriptionAPIBaseURL,
	}
}

func (c *DescriptionClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *DescriptionClient) SetIngestionRunID(runID int) {
	c.httpClient.SetIngestionRunID(runID)
}

func (c *DescriptionClient) FetchDescription(ctx context.Context, noticeID string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("api_key", c.apiKey)
	q.Set("noticeid", noticeID)
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(ctx, u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var descResp DescriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&descResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return descResp.Description, nil
}

func (c *DescriptionClient) FetchDescriptionsForNotices(
	ctx context.Context,
	db *database.DB,
	noticeIDs []string,
) (*DescriptionResult, error) {
	result := &DescriptionResult{}

	for i, noticeID := range noticeIDs {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		description, err := c.FetchDescription(ctx, noticeID)
		if err != nil {
			c.logger.Warn("failed to fetch description",
				"notice_id", noticeID,
				"error", err,
			)
			result.Errors++
		} else {
			if err := db.UpdateOpportunityDescription(ctx, noticeID, description); err != nil {
				c.logger.Error("failed to update description in DB",
					"notice_id", noticeID,
					"error", err,
				)
				result.Errors++
			} else {
				result.Fetched++
				c.logger.Debug("fetched description", "notice_id", noticeID)
			}
		}

		if i < len(noticeIDs)-1 {
			time.Sleep(DescriptionDelay)
		}

		if (i+1)%100 == 0 {
			c.logger.Info("description fetch progress",
				"completed", i+1,
				"total", len(noticeIDs),
				"fetched", result.Fetched,
				"errors", result.Errors,
			)
		}
	}

	c.logger.Info("description fetch complete",
		"total", len(noticeIDs),
		"fetched", result.Fetched,
		"errors", result.Errors,
	)

	return result, nil
}

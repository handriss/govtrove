package apiprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/ingestion/internal/config"
	"github.com/handriss/govtrove/ingestion/internal/database"
	"github.com/handriss/govtrove/ingestion/internal/samgov"
)

type ProbeResult struct {
	TotalRecords   int
	ResponseTimeMs int
}

func RunProbe(ctx context.Context, cfg *config.APIProbeConfig, db *database.DB, s3Client *s3.Client, logger *slog.Logger) (*ProbeResult, error) {
	et, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("load ET timezone: %w", err)
	}

	now := time.Now().In(et)
	yesterday := now.AddDate(0, 0, -1)
	postedFrom := yesterday.Format("01/02/2006")
	postedTo := now.Format("01/02/2006")

	u, err := url.Parse(samgov.SearchAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("api_key", cfg.SAMAPIKey)
	q.Set("postedFrom", postedFrom)
	q.Set("postedTo", postedTo)
	q.Set("limit", "10")
	u.RawQuery = q.Encode()

	logger.Info("probing SAM.gov API", "posted_from", postedFrom, "posted_to", postedTo)

	httpClient := samgov.NewTrackedHTTPClient(db, logger)
	resp, err := httpClient.Get(ctx, u.String())

	record := &database.APIUpdateProbeRecord{
		PostedFrom: &postedFrom,
	}

	if err != nil {
		errMsg := err.Error()
		record.ErrorMessage = &errMsg
		if _, dbErr := db.InsertAPIUpdateProbe(ctx, record); dbErr != nil {
			logger.Warn("failed to log probe error", "error", dbErr)
		}
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	record.HTTPStatus = &resp.StatusCode
	record.ResponseTimeMs = &resp.ResponseTimeMs

	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		record.ErrorMessage = &errMsg
		if _, dbErr := db.InsertAPIUpdateProbe(ctx, record); dbErr != nil {
			logger.Warn("failed to log probe error", "error", dbErr)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf("read body: %s", err)
		record.ErrorMessage = &errMsg
		if _, dbErr := db.InsertAPIUpdateProbe(ctx, record); dbErr != nil {
			logger.Warn("failed to log probe error", "error", dbErr)
		}
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var searchResp samgov.APISearchResponse
	if err := json.Unmarshal(rawBody, &searchResp); err != nil {
		errMsg := fmt.Sprintf("decode error: %s", err)
		record.ErrorMessage = &errMsg
		if _, dbErr := db.InsertAPIUpdateProbe(ctx, record); dbErr != nil {
			logger.Warn("failed to log probe error", "error", dbErr)
		}
		return nil, fmt.Errorf("decode response: %w", err)
	}

	record.TotalRecords = &searchResp.TotalRecords
	if len(searchResp.OpportunitiesData) > 0 {
		first := searchResp.OpportunitiesData[0]
		record.SampleNoticeID = &first.NoticeID
		record.SamplePostedDate = &first.PostedDate
	}

	id, err := db.InsertAPIUpdateProbe(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("insert probe record: %w", err)
	}

	// Save raw response to S3
	if cfg.S3ArchiveEnabled && s3Client != nil {
		ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
		s3Key := fmt.Sprintf("raw/api/probe/%s.json", ts)
		_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(cfg.S3Bucket),
			Key:         aws.String(s3Key),
			Body:        bytes.NewReader(rawBody),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			logger.Warn("failed to upload probe response to S3", "error", err)
		} else {
			logger.Info("probe response saved to S3", "key", s3Key, "size_bytes", len(rawBody))
		}
	}

	logger.Info("probe complete",
		"id", id,
		"total_records", searchResp.TotalRecords,
		"response_time_ms", resp.ResponseTimeMs,
		"posted_from", postedFrom,
	)

	return &ProbeResult{
		TotalRecords:   searchResp.TotalRecords,
		ResponseTimeMs: resp.ResponseTimeMs,
	}, nil
}

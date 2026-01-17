package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/opscout/ingestion/internal/config"
	"github.com/opscout/ingestion/internal/database"
)

type Service struct {
	cfg    *config.Config
	db     *database.DB
	sns    *sns.Client
	logger *slog.Logger
}

func New(cfg *config.Config, db *database.DB, snsClient *sns.Client, logger *slog.Logger) *Service {
	return &Service{
		cfg:    cfg,
		db:     db,
		sns:    snsClient,
		logger: logger,
	}
}

// TODO: Replace example data with actual SAM.gov API calls
func (s *Service) Run(ctx context.Context) error {
	startTime := time.Now()
	s.logger.Info("starting ingestion")

	runID, err := s.db.CreateIngestionRun(ctx)
	if err != nil {
		s.logger.Error("failed to create ingestion run", "error", err)
		return err
	}
	s.logger.Info("created ingestion run", "run_id", runID)

	var (
		fetched  int
		inserted int
		updated  int
		failed   int
	)

	exampleOpportunities := s.getExampleOpportunities()
	fetched = len(exampleOpportunities)

	for _, opp := range exampleOpportunities {
		wasInserted, err := s.db.UpsertOpportunity(ctx, opp)
		if err != nil {
			s.logger.Error("failed to upsert opportunity", "notice_id", opp.NoticeID, "error", err)
			failed++
			continue
		}
		if wasInserted {
			inserted++
			s.logger.Debug("inserted opportunity", "notice_id", opp.NoticeID)
		} else {
			updated++
			s.logger.Debug("updated opportunity", "notice_id", opp.NoticeID)
		}
	}

	durationMs := int(time.Since(startTime).Milliseconds())

	if err := s.db.CompleteIngestionRun(ctx, runID, fetched, inserted, updated, failed, durationMs); err != nil {
		s.logger.Error("failed to complete ingestion run", "error", err)
		return err
	}

	// This log line is picked up by CloudWatch metric filter for alerting
	s.logger.Info("ingestion completed",
		"run_id", runID,
		"fetched", fetched,
		"inserted", inserted,
		"updated", updated,
		"failed", failed,
		"duration_ms", durationMs,
	)

	if err := s.sendNotification(ctx, runID, fetched, inserted, updated, failed, durationMs); err != nil {
		// Don't fail the run for notification errors
		s.logger.Warn("failed to send notification", "error", err)
	}

	total, err := s.db.CountOpportunities(ctx)
	if err != nil {
		s.logger.Warn("failed to count opportunities", "error", err)
	} else {
		s.logger.Info("total opportunities in database", "count", total)
	}

	return nil
}

func (s *Service) sendNotification(ctx context.Context, runID, fetched, inserted, updated, failed, durationMs int) error {
	if s.sns == nil || s.cfg.SNSTopicARN == "" {
		s.logger.Debug("SNS not configured, skipping notification")
		return nil
	}

	status := "SUCCESS"
	if failed > 0 {
		status = "PARTIAL_FAILURE"
	}

	message := fmt.Sprintf(`OpScout Ingestion Report
========================
Status: %s
Run ID: %d

Records Fetched:  %d
Records Inserted: %d
Records Updated:  %d
Records Failed:   %d

Duration: %dms
`, status, runID, fetched, inserted, updated, failed, durationMs)

	_, err := s.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(s.cfg.SNSTopicARN),
		Subject:  aws.String(fmt.Sprintf("OpScout Ingestion: %s", status)),
		Message:  aws.String(message),
	})
	return err
}

// TODO: Replace with actual SAM.gov API integration
func (s *Service) getExampleOpportunities() []*database.Opportunity {
	now := time.Now()
	deadline := now.AddDate(0, 0, 30)

	rawJSON1, _ := json.Marshal(map[string]any{
		"noticeId": "EXAMPLE-001",
		"title":    "Example IT Services Contract",
		"_note":    "This is example data for testing",
	})

	rawJSON2, _ := json.Marshal(map[string]any{
		"noticeId": "EXAMPLE-002",
		"title":    "Example Cybersecurity Assessment",
		"_note":    "This is example data for testing",
	})

	return []*database.Opportunity{
		{
			NoticeID:            "EXAMPLE-001",
			SolicitationNumber:  "W91QUZ-24-R-0001",
			Title:               "Example IT Services Contract",
			Type:                "Solicitation",
			BaseType:            "Solicitation",
			PostedDate:          &now,
			ResponseDeadline:    &deadline,
			SetAsideCode:        "SBA",
			SetAsideDescription: "Total Small Business Set-Aside",
			NAICSCode:           "541512",
			ClassificationCode:  "D302",
			OrganizationType:    "OFFICE",
			FullParentPathName:  "DEPT OF DEFENSE.DEPT OF THE ARMY.EXAMPLE CONTRACTING OFFICE",
			FullParentPathCode:  "097.021.EXAMPLE",
			Active:              true,
			RawJSON:             rawJSON1,
		},
		{
			NoticeID:            "EXAMPLE-002",
			SolicitationNumber:  "70SBUR24R00000001",
			Title:               "Example Cybersecurity Assessment Services",
			Type:                "Presolicitation",
			BaseType:            "Presolicitation",
			PostedDate:          &now,
			ResponseDeadline:    nil,
			SetAsideCode:        "8A",
			SetAsideDescription: "8(a) Set-Aside",
			NAICSCode:           "541519",
			ClassificationCode:  "D399",
			OrganizationType:    "OFFICE",
			FullParentPathName:  "HOMELAND SECURITY, DEPARTMENT OF.EXAMPLE COMPONENT",
			FullParentPathCode:  "070.EXAMPLE",
			Active:              true,
			RawJSON:             rawJSON2,
		},
	}
}

package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/opscout/ingestion/internal/config"
	"github.com/opscout/ingestion/internal/database"
	"github.com/opscout/ingestion/internal/samgov"
)

type Service struct {
	cfg               *config.Config
	db                *database.DB
	sns               *sns.Client
	csvClient         *samgov.CSVClient
	apiClient         *samgov.APIClient
	descriptionClient *samgov.DescriptionClient
	logger            *slog.Logger
}

type IngestionStats struct {
	CSVFetched          int
	APIFetched          int
	CSVOnly             int
	APIOnly             int
	InBoth              int
	Inserted            int
	Updated             int
	Failed              int
	DescriptionsFetched int
	DataMismatches      int
}

func New(cfg *config.Config, db *database.DB, snsClient *sns.Client, logger *slog.Logger) *Service {
	s := &Service{
		cfg:       cfg,
		db:        db,
		sns:       snsClient,
		csvClient: samgov.NewCSVClient(db, logger),
		logger:    logger,
	}

	s.apiClient = samgov.NewAPIClient(db, logger, cfg.SAMAPIKey)
	s.descriptionClient = samgov.NewDescriptionClient(db, logger, cfg.SAMAPIKey)

	if cfg.IsMockMode() {
		s.apiClient.SetBaseURL(cfg.MockAPIURL + "/prod/opportunities/v2/search")
		s.descriptionClient.SetBaseURL(cfg.MockAPIURL + "/prod/opportunities/v1/noticedesc")
		logger.Info("using mock API", "base_url", cfg.MockAPIURL)
	}

	return s
}

func (s *Service) Run(ctx context.Context) error {
	startTime := time.Now()
	s.logger.Info("starting ingestion", "mode", s.cfg.IngestionMode)

	runID, err := s.db.CreateIngestionRunWithMode(ctx, string(s.cfg.IngestionMode), s.cfg.LookbackDays)
	if err != nil {
		s.logger.Error("failed to create ingestion run", "error", err)
		return err
	}
	s.logger.Info("created ingestion run", "run_id", runID, "mode", s.cfg.IngestionMode)

	s.csvClient.SetIngestionRunID(runID)
	s.apiClient.SetIngestionRunID(runID)
	s.descriptionClient.SetIngestionRunID(runID)

	stats := &IngestionStats{}

	// API disabled until we get higher rate limits (currently 1000 requests/day).
	// Once approved, remove this block and uncomment the switch below.
	const apiEnabled = false
	if !apiEnabled {
		s.logger.Info("API disabled, using CSV-only mode")
		err = s.runCSVOnly(ctx, runID, startTime, stats)
	} else {
		switch s.cfg.IngestionMode {
		case config.ModeCSVOnly:
			err = s.runCSVOnly(ctx, runID, startTime, stats)
		case config.ModeIncremental:
			err = s.runIncremental(ctx, runID, startTime, stats)
		default:
			err = s.runFull(ctx, runID, startTime, stats)
		}
	}

	if err != nil {
		return err
	}

	total, err := s.db.CountOpportunities(ctx)
	if err != nil {
		s.logger.Warn("failed to count opportunities", "error", err)
	} else {
		s.logger.Info("total opportunities in database", "count", total)
	}

	return nil
}

func (s *Service) runCSVOnly(ctx context.Context, runID int, startTime time.Time, stats *IngestionStats) error {
	s.logger.Info("running CSV-only mode (fast backfill)")

	_, err := s.fetchAndSaveCSV(ctx, runID, stats)
	if err != nil {
		s.failRun(ctx, runID, startTime, err)
		return err
	}

	durationMs := int(time.Since(startTime).Milliseconds())
	if err := s.db.CompleteIngestionRun(ctx, runID, stats.CSVFetched, stats.Inserted, stats.Updated, stats.Failed, durationMs); err != nil {
		s.logger.Error("failed to complete ingestion run", "error", err)
		return err
	}

	s.logFinalStats(runID, stats, durationMs)

	if err := s.sendNotification(ctx, runID, stats, durationMs); err != nil {
		s.logger.Warn("failed to send notification", "error", err)
	}

	return nil
}

func (s *Service) runIncremental(ctx context.Context, runID int, startTime time.Time, stats *IngestionStats) error {
	s.logger.Info("running incremental mode", "lookback_days", s.cfg.LookbackDays)

	since := time.Now().AddDate(0, 0, -s.cfg.LookbackDays)
	result, err := s.apiClient.FetchOpportunitiesSince(ctx, since, s.cfg.RecordLimit)
	if err != nil {
		s.failRun(ctx, runID, startTime, err)
		return err
	}

	stats.APIFetched = len(result.Opportunities)
	s.logger.Info("API incremental fetch complete",
		"fetched", stats.APIFetched,
		"since", since.Format("2006-01-02"),
	)

	for _, opp := range result.Opportunities {
		opp.DataSource = "api"
		opp.IngestionRunID = &runID

		wasInserted, err := s.db.UpsertOpportunity(ctx, opp)
		if err != nil {
			s.logger.Error("failed to upsert API opportunity", "notice_id", opp.NoticeID, "error", err)
			stats.Failed++
			continue
		}
		if wasInserted {
			stats.Inserted++
		} else {
			stats.Updated++
		}
	}

	if !s.cfg.SkipDescriptions && len(result.Opportunities) > 0 {
		noticeIDs := make([]string, 0, len(result.Opportunities))
		for _, opp := range result.Opportunities {
			noticeIDs = append(noticeIDs, opp.NoticeID)
		}
		s.fetchDescriptions(ctx, noticeIDs, stats)
	}

	durationMs := int(time.Since(startTime).Milliseconds())
	if err := s.db.CompleteIngestionRun(ctx, runID, stats.APIFetched, stats.Inserted, stats.Updated, stats.Failed, durationMs); err != nil {
		s.logger.Error("failed to complete ingestion run", "error", err)
		return err
	}

	s.logFinalStats(runID, stats, durationMs)

	if err := s.sendNotification(ctx, runID, stats, durationMs); err != nil {
		s.logger.Warn("failed to send notification", "error", err)
	}

	return nil
}

func (s *Service) runFull(ctx context.Context, runID int, startTime time.Time, stats *IngestionStats) error {
	// PHASE 1: Download and save ALL CSV data
	s.logger.Info("PHASE 1: Fetching CSV data")
	csvOpps, err := s.fetchAndSaveCSV(ctx, runID, stats)
	if err != nil {
		s.failRun(ctx, runID, startTime, err)
		return err
	}
	csvNoticeIDs := s.buildNoticeIDSet(csvOpps)

	// PHASE 2: Fetch ALL opportunities from API
	var apiOpps map[string]*database.Opportunity
	if !s.cfg.SkipAPI {
		s.logger.Info("PHASE 2: Fetching API data")
		apiOpps, err = s.fetchAPI(ctx, stats)
		if err != nil {
			s.logger.Error("API fetch failed, continuing with CSV data only", "error", err)
		}
	} else {
		s.logger.Info("PHASE 2: Skipping API fetch (SKIP_API=true)")
	}

	// PHASE 3: Cross-reference and identify API-only opportunities
	var apiOnlyIDs []string
	if apiOpps != nil {
		s.logger.Info("PHASE 3: Cross-referencing data sources")
		apiOnlyIDs = s.crossReference(ctx, csvNoticeIDs, apiOpps, stats)
	} else {
		s.logger.Info("PHASE 3: Skipping cross-reference (no API data)")
		stats.CSVOnly = len(csvNoticeIDs)
	}

	// PHASE 4: Save API-only opportunities
	if len(apiOnlyIDs) > 0 {
		s.logger.Info("PHASE 4: Saving API-only opportunities", "count", len(apiOnlyIDs))
		s.saveAPIOnlyOpportunities(ctx, apiOpps, apiOnlyIDs, runID, stats)
	} else {
		s.logger.Info("PHASE 4: No API-only opportunities to save")
	}

	// PHASE 5: Fetch descriptions for API-only opportunities
	if !s.cfg.SkipDescriptions && len(apiOnlyIDs) > 0 {
		s.logger.Info("PHASE 5: Fetching descriptions for API-only opportunities", "count", len(apiOnlyIDs))
		s.fetchDescriptions(ctx, apiOnlyIDs, stats)
	} else if s.cfg.SkipDescriptions {
		s.logger.Info("PHASE 5: Skipping description fetch (SKIP_DESCRIPTIONS=true)")
	} else {
		s.logger.Info("PHASE 5: No descriptions to fetch")
	}

	durationMs := int(time.Since(startTime).Milliseconds())
	totalFetched := stats.CSVFetched + stats.APIOnly

	if err := s.db.CompleteIngestionRun(ctx, runID, totalFetched, stats.Inserted, stats.Updated, stats.Failed, durationMs); err != nil {
		s.logger.Error("failed to complete ingestion run", "error", err)
		return err
	}

	s.logFinalStats(runID, stats, durationMs)

	if err := s.sendNotification(ctx, runID, stats, durationMs); err != nil {
		s.logger.Warn("failed to send notification", "error", err)
	}

	return nil
}

func (s *Service) fetchAndSaveCSV(ctx context.Context, runID int, stats *IngestionStats) ([]*database.Opportunity, error) {
	result, err := s.csvClient.FetchOpportunities(ctx, s.cfg.RecordLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CSV: %w", err)
	}

	stats.CSVFetched = result.ParsedRows
	s.logger.Info("CSV download complete", "parsed", result.ParsedRows, "total_rows", result.TotalRows)

	for _, opp := range result.Opportunities {
		opp.DataSource = "csv"
		opp.IngestionRunID = &runID

		wasInserted, err := s.db.UpsertOpportunity(ctx, opp)
		if err != nil {
			s.logger.Error("failed to upsert CSV opportunity", "notice_id", opp.NoticeID, "error", err)
			stats.Failed++
			continue
		}
		if wasInserted {
			stats.Inserted++
			s.logger.Debug("inserted CSV opportunity", "notice_id", opp.NoticeID)
		} else {
			stats.Updated++
			s.logger.Debug("updated CSV opportunity", "notice_id", opp.NoticeID)
		}
	}

	s.logger.Info("CSV phase complete",
		"fetched", stats.CSVFetched,
		"inserted", stats.Inserted,
		"updated", stats.Updated,
		"failed", stats.Failed,
	)

	return result.Opportunities, nil
}

func (s *Service) fetchAPI(ctx context.Context, stats *IngestionStats) (map[string]*database.Opportunity, error) {
	result, err := s.apiClient.FetchAllOpportunities(ctx, s.cfg.RecordLimit)
	if err != nil {
		return nil, err
	}

	stats.APIFetched = len(result.Opportunities)
	s.logger.Info("API fetch complete",
		"fetched", stats.APIFetched,
		"total_available", result.TotalRecords,
		"pages", result.PagesFetched,
	)

	return s.buildNoticeIDMap(result.Opportunities), nil
}

func (s *Service) buildNoticeIDSet(opps []*database.Opportunity) map[string]bool {
	set := make(map[string]bool, len(opps))
	for _, opp := range opps {
		set[opp.NoticeID] = true
	}
	return set
}

func (s *Service) buildNoticeIDMap(opps []*database.Opportunity) map[string]*database.Opportunity {
	m := make(map[string]*database.Opportunity, len(opps))
	for _, opp := range opps {
		m[opp.NoticeID] = opp
	}
	return m
}

func (s *Service) crossReference(
	ctx context.Context,
	csvIDs map[string]bool,
	apiOpps map[string]*database.Opportunity,
	stats *IngestionStats,
) []string {
	var apiOnlyIDs []string

	for noticeID, apiOpp := range apiOpps {
		if csvIDs[noticeID] {
			stats.InBoth++

			if err := s.db.UpdateOpportunitySource(ctx, noticeID, "csv+api"); err != nil {
				s.logger.Warn("failed to update source", "notice_id", noticeID, "error", err)
			}

			if s.cfg.VerboseLogging {
				s.compareOpportunity(ctx, noticeID, apiOpp, stats)
			}
		} else {
			stats.APIOnly++
			apiOnlyIDs = append(apiOnlyIDs, noticeID)
		}
	}

	for noticeID := range csvIDs {
		if _, inAPI := apiOpps[noticeID]; !inAPI {
			stats.CSVOnly++
		}
	}

	s.logger.Info("cross-reference complete",
		"csv_only", stats.CSVOnly,
		"api_only", stats.APIOnly,
		"in_both", stats.InBoth,
		"mismatches", stats.DataMismatches,
	)

	return apiOnlyIDs
}

func (s *Service) compareOpportunity(ctx context.Context, noticeID string, apiOpp *database.Opportunity, stats *IngestionStats) {
	csvOpp, err := s.db.GetOpportunityByNoticeID(ctx, noticeID)
	if err != nil {
		s.logger.Debug("could not fetch CSV opportunity for comparison", "notice_id", noticeID, "error", err)
		return
	}

	var diffs []string

	if csvOpp.Title != apiOpp.Title {
		diffs = append(diffs, fmt.Sprintf("title: csv=%q api=%q", csvOpp.Title, apiOpp.Title))
	}
	if csvOpp.SolicitationNumber != apiOpp.SolicitationNumber {
		diffs = append(diffs, fmt.Sprintf("sol_number: csv=%q api=%q", csvOpp.SolicitationNumber, apiOpp.SolicitationNumber))
	}
	if csvOpp.Type != apiOpp.Type {
		diffs = append(diffs, fmt.Sprintf("type: csv=%q api=%q", csvOpp.Type, apiOpp.Type))
	}
	if csvOpp.SetAsideCode != apiOpp.SetAsideCode {
		diffs = append(diffs, fmt.Sprintf("set_aside: csv=%q api=%q", csvOpp.SetAsideCode, apiOpp.SetAsideCode))
	}

	if len(diffs) > 0 {
		stats.DataMismatches++
		s.logger.Warn("data mismatch between CSV and API",
			"notice_id", noticeID,
			"differences", diffs,
		)
	}
}

func (s *Service) saveAPIOnlyOpportunities(
	ctx context.Context,
	apiOpps map[string]*database.Opportunity,
	apiOnlyIDs []string,
	runID int,
	stats *IngestionStats,
) {
	for _, noticeID := range apiOnlyIDs {
		opp := apiOpps[noticeID]
		opp.DataSource = "api"
		opp.IngestionRunID = &runID

		wasInserted, err := s.db.UpsertOpportunity(ctx, opp)
		if err != nil {
			s.logger.Error("failed to upsert API opportunity", "notice_id", noticeID, "error", err)
			stats.Failed++
			continue
		}
		if wasInserted {
			stats.Inserted++
			s.logger.Debug("inserted API opportunity", "notice_id", noticeID)
		} else {
			stats.Updated++
			s.logger.Debug("updated API opportunity", "notice_id", noticeID)
		}
	}
}

func (s *Service) fetchDescriptions(ctx context.Context, noticeIDs []string, stats *IngestionStats) {
	result, err := s.descriptionClient.FetchDescriptionsForNotices(ctx, s.db, noticeIDs)
	if err != nil {
		s.logger.Error("description fetch interrupted", "error", err)
	}
	stats.DescriptionsFetched = result.Fetched
	stats.Failed += result.Errors
}

func (s *Service) failRun(ctx context.Context, runID int, startTime time.Time, err error) {
	durationMs := int(time.Since(startTime).Milliseconds())
	s.db.FailIngestionRun(ctx, runID, err.Error(), durationMs)
	s.logger.Error("ingestion run failed", "error", err, "duration_ms", durationMs)
}

func (s *Service) logFinalStats(runID int, stats *IngestionStats, durationMs int) {
	s.logger.Info("ingestion completed",
		"run_id", runID,
		"csv_fetched", stats.CSVFetched,
		"api_fetched", stats.APIFetched,
		"csv_only", stats.CSVOnly,
		"api_only", stats.APIOnly,
		"in_both", stats.InBoth,
		"inserted", stats.Inserted,
		"updated", stats.Updated,
		"failed", stats.Failed,
		"descriptions_fetched", stats.DescriptionsFetched,
		"data_mismatches", stats.DataMismatches,
		"duration_ms", durationMs,
	)
}

func (s *Service) sendNotification(ctx context.Context, runID int, stats *IngestionStats, durationMs int) error {
	if s.sns == nil || s.cfg.SNSTopicARN == "" {
		s.logger.Debug("SNS not configured, skipping notification")
		return nil
	}

	status := "SUCCESS"
	if stats.Failed > 0 {
		status = "PARTIAL_FAILURE"
	}

	message := fmt.Sprintf(`OpScout Ingestion Report
========================
Status: %s
Run ID: %d

CSV Records:      %d
API Records:      %d
CSV Only:         %d
API Only:         %d
In Both Sources:  %d

Records Inserted: %d
Records Updated:  %d
Records Failed:   %d

Descriptions:     %d
Data Mismatches:  %d

Duration: %dms
`, status, runID,
		stats.CSVFetched, stats.APIFetched,
		stats.CSVOnly, stats.APIOnly, stats.InBoth,
		stats.Inserted, stats.Updated, stats.Failed,
		stats.DescriptionsFetched, stats.DataMismatches,
		durationMs)

	_, err := s.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(s.cfg.SNSTopicARN),
		Subject:  aws.String(fmt.Sprintf("OpScout Ingestion: %s", status)),
		Message:  aws.String(message),
	})
	return err
}

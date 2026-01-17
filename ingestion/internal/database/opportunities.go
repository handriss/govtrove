package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Opportunity struct {
	NoticeID            string
	SolicitationNumber  string
	Title               string
	DescriptionURL      string
	UILink              string
	Type                string
	BaseType            string
	PostedDate          *time.Time
	ResponseDeadline    *time.Time
	SetAsideCode        string
	SetAsideDescription string
	NAICSCode           string
	ClassificationCode  string
	OrganizationType    string
	FullParentPathName  string
	FullParentPathCode  string
	Active              bool
	RawJSON             json.RawMessage
}

type IngestionRun struct {
	ID              int
	StartedAt       time.Time
	CompletedAt     *time.Time
	Status          string
	RecordsFetched  int
	RecordsInserted int
	RecordsUpdated  int
	RecordsFailed   int
	ErrorMessage    string
	DurationMs      int
}

func (db *DB) CreateIngestionRun(ctx context.Context) (int, error) {
	var id int
	err := db.pool.QueryRow(ctx,
		`INSERT INTO ingestion_runs (status) VALUES ('running') RETURNING id`,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create ingestion run: %w", err)
	}
	return id, nil
}

func (db *DB) CompleteIngestionRun(ctx context.Context, id int, fetched, inserted, updated, failed int, durationMs int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE ingestion_runs
		SET status = 'completed',
		    completed_at = NOW(),
		    records_fetched = $2,
		    records_inserted = $3,
		    records_updated = $4,
		    records_failed = $5,
		    duration_ms = $6
		WHERE id = $1
	`, id, fetched, inserted, updated, failed, durationMs)
	if err != nil {
		return fmt.Errorf("failed to complete ingestion run: %w", err)
	}
	return nil
}

func (db *DB) FailIngestionRun(ctx context.Context, id int, errMsg string, durationMs int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE ingestion_runs
		SET status = 'failed',
		    completed_at = NOW(),
		    error_message = $2,
		    duration_ms = $3
		WHERE id = $1
	`, id, errMsg, durationMs)
	if err != nil {
		return fmt.Errorf("failed to mark ingestion run as failed: %w", err)
	}
	return nil
}

// Returns true if inserted, false if updated
func (db *DB) UpsertOpportunity(ctx context.Context, opp *Opportunity) (inserted bool, err error) {
	var resultID int
	err = db.pool.QueryRow(ctx, `
		INSERT INTO opportunities (
			notice_id,
			solicitation_number,
			title,
			description_url,
			ui_link,
			type,
			base_type,
			posted_date,
			response_deadline,
			set_aside_code,
			set_aside_description,
			naics_code,
			classification_code,
			organization_type,
			full_parent_path_name,
			full_parent_path_code,
			active,
			raw_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (notice_id) DO UPDATE SET
			solicitation_number = EXCLUDED.solicitation_number,
			title = EXCLUDED.title,
			description_url = EXCLUDED.description_url,
			ui_link = EXCLUDED.ui_link,
			type = EXCLUDED.type,
			base_type = EXCLUDED.base_type,
			posted_date = EXCLUDED.posted_date,
			response_deadline = EXCLUDED.response_deadline,
			set_aside_code = EXCLUDED.set_aside_code,
			set_aside_description = EXCLUDED.set_aside_description,
			naics_code = EXCLUDED.naics_code,
			classification_code = EXCLUDED.classification_code,
			organization_type = EXCLUDED.organization_type,
			full_parent_path_name = EXCLUDED.full_parent_path_name,
			full_parent_path_code = EXCLUDED.full_parent_path_code,
			active = EXCLUDED.active,
			raw_json = EXCLUDED.raw_json
		RETURNING id, (xmax = 0) AS inserted
	`, opp.NoticeID, opp.SolicitationNumber, opp.Title, opp.DescriptionURL,
		opp.UILink, opp.Type, opp.BaseType, opp.PostedDate, opp.ResponseDeadline,
		opp.SetAsideCode, opp.SetAsideDescription, opp.NAICSCode, opp.ClassificationCode,
		opp.OrganizationType, opp.FullParentPathName, opp.FullParentPathCode,
		opp.Active, opp.RawJSON,
	).Scan(&resultID, &inserted)

	if err != nil {
		return false, fmt.Errorf("failed to upsert opportunity %s: %w", opp.NoticeID, err)
	}
	return inserted, nil
}

func (db *DB) CountOpportunities(ctx context.Context) (int, error) {
	var count int
	err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM opportunities`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count opportunities: %w", err)
	}
	return count, nil
}

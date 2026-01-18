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
	Description         string
	DescriptionURL      string
	UILink              string
	AdditionalInfoLink  string
	Type                string
	BaseType            string
	PostedDate          *time.Time
	ResponseDeadline    *time.Time
	ArchiveDate         *time.Time
	ArchiveType         string
	SetAsideCode        string
	SetAsideDescription string
	NAICSCode           string
	NAICSCodes          []string
	ClassificationCode  string
	OrganizationType    string
	FullParentPathName  string
	FullParentPathCode  string

	Department string
	SubTier    string
	Office     string
	CGAC       string
	FPDSCode   string
	AACCode    string

	PopStreetAddress string
	PopCity          string
	PopStateCode     string
	PopZip           string
	PopCountryCode   string

	OfficeCity        string
	OfficeState       string
	OfficeZip         string
	OfficeCountryCode string

	AwardNumber string
	AwardAmount *float64
	AwardDate   *time.Time
	AwardeeName string

	PrimaryContactTitle    string
	PrimaryContactFullname string
	PrimaryContactEmail    string
	PrimaryContactPhone    string
	PrimaryContactFax      string

	SecondaryContactTitle    string
	SecondaryContactFullname string
	SecondaryContactEmail    string
	SecondaryContactPhone    string
	SecondaryContactFax      string

	Active         bool
	DataSource     string
	IngestionRunID *int
	RawJSON        json.RawMessage
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
			description,
			description_url,
			ui_link,
			additional_info_link,
			type,
			base_type,
			posted_date,
			response_deadline,
			archive_date,
			archive_type,
			set_aside_code,
			set_aside_description,
			naics_code,
			naics_codes,
			classification_code,
			organization_type,
			full_parent_path_name,
			full_parent_path_code,
			department,
			sub_tier,
			office,
			cgac,
			fpds_code,
			aac_code,
			pop_street_address,
			pop_city_name,
			pop_state_code,
			pop_zip,
			pop_country_code,
			office_city,
			office_state,
			office_zip,
			office_country_code,
			award_number,
			award_amount,
			award_date,
			awardee_name,
			primary_contact_title,
			primary_contact_fullname,
			primary_contact_email,
			primary_contact_phone,
			primary_contact_fax,
			secondary_contact_title,
			secondary_contact_fullname,
			secondary_contact_email,
			secondary_contact_phone,
			secondary_contact_fax,
			active,
			data_source,
			ingestion_run_id,
			raw_json
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
			$41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
			$51, $52, $53, $54
		)
		ON CONFLICT (notice_id) DO UPDATE SET
			solicitation_number = EXCLUDED.solicitation_number,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			description_url = EXCLUDED.description_url,
			ui_link = EXCLUDED.ui_link,
			additional_info_link = EXCLUDED.additional_info_link,
			type = EXCLUDED.type,
			base_type = EXCLUDED.base_type,
			posted_date = EXCLUDED.posted_date,
			response_deadline = EXCLUDED.response_deadline,
			archive_date = EXCLUDED.archive_date,
			archive_type = EXCLUDED.archive_type,
			set_aside_code = EXCLUDED.set_aside_code,
			set_aside_description = EXCLUDED.set_aside_description,
			naics_code = EXCLUDED.naics_code,
			naics_codes = EXCLUDED.naics_codes,
			classification_code = EXCLUDED.classification_code,
			organization_type = EXCLUDED.organization_type,
			full_parent_path_name = EXCLUDED.full_parent_path_name,
			full_parent_path_code = EXCLUDED.full_parent_path_code,
			department = EXCLUDED.department,
			sub_tier = EXCLUDED.sub_tier,
			office = EXCLUDED.office,
			cgac = EXCLUDED.cgac,
			fpds_code = EXCLUDED.fpds_code,
			aac_code = EXCLUDED.aac_code,
			pop_street_address = EXCLUDED.pop_street_address,
			pop_city_name = EXCLUDED.pop_city_name,
			pop_state_code = EXCLUDED.pop_state_code,
			pop_zip = EXCLUDED.pop_zip,
			pop_country_code = EXCLUDED.pop_country_code,
			office_city = EXCLUDED.office_city,
			office_state = EXCLUDED.office_state,
			office_zip = EXCLUDED.office_zip,
			office_country_code = EXCLUDED.office_country_code,
			award_number = EXCLUDED.award_number,
			award_amount = EXCLUDED.award_amount,
			award_date = EXCLUDED.award_date,
			awardee_name = EXCLUDED.awardee_name,
			primary_contact_title = EXCLUDED.primary_contact_title,
			primary_contact_fullname = EXCLUDED.primary_contact_fullname,
			primary_contact_email = EXCLUDED.primary_contact_email,
			primary_contact_phone = EXCLUDED.primary_contact_phone,
			primary_contact_fax = EXCLUDED.primary_contact_fax,
			secondary_contact_title = EXCLUDED.secondary_contact_title,
			secondary_contact_fullname = EXCLUDED.secondary_contact_fullname,
			secondary_contact_email = EXCLUDED.secondary_contact_email,
			secondary_contact_phone = EXCLUDED.secondary_contact_phone,
			secondary_contact_fax = EXCLUDED.secondary_contact_fax,
			active = EXCLUDED.active,
			data_source = EXCLUDED.data_source,
			ingestion_run_id = EXCLUDED.ingestion_run_id,
			raw_json = EXCLUDED.raw_json
		RETURNING id, (xmax = 0) AS inserted
	`,
		opp.NoticeID, opp.SolicitationNumber, opp.Title, opp.Description,
		opp.DescriptionURL, opp.UILink, opp.AdditionalInfoLink,
		opp.Type, opp.BaseType, opp.PostedDate, opp.ResponseDeadline,
		opp.ArchiveDate, opp.ArchiveType, opp.SetAsideCode, opp.SetAsideDescription,
		opp.NAICSCode, opp.NAICSCodes, opp.ClassificationCode, opp.OrganizationType,
		opp.FullParentPathName, opp.FullParentPathCode,
		opp.Department, opp.SubTier, opp.Office, opp.CGAC, opp.FPDSCode, opp.AACCode,
		opp.PopStreetAddress, opp.PopCity, opp.PopStateCode, opp.PopZip, opp.PopCountryCode,
		opp.OfficeCity, opp.OfficeState, opp.OfficeZip, opp.OfficeCountryCode,
		opp.AwardNumber, opp.AwardAmount, opp.AwardDate, opp.AwardeeName,
		opp.PrimaryContactTitle, opp.PrimaryContactFullname, opp.PrimaryContactEmail,
		opp.PrimaryContactPhone, opp.PrimaryContactFax,
		opp.SecondaryContactTitle, opp.SecondaryContactFullname, opp.SecondaryContactEmail,
		opp.SecondaryContactPhone, opp.SecondaryContactFax,
		opp.Active, opp.DataSource, opp.IngestionRunID, opp.RawJSON,
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

func (db *DB) UpdateOpportunityDescription(ctx context.Context, noticeID, description string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE opportunities
		SET description = $2, updated_at = NOW()
		WHERE notice_id = $1
	`, noticeID, description)
	if err != nil {
		return fmt.Errorf("failed to update description for %s: %w", noticeID, err)
	}
	return nil
}

func (db *DB) UpdateOpportunitySource(ctx context.Context, noticeID, source string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE opportunities
		SET data_source = $2
		WHERE notice_id = $1
	`, noticeID, source)
	if err != nil {
		return fmt.Errorf("failed to update source for %s: %w", noticeID, err)
	}
	return nil
}

func (db *DB) GetOpportunityByNoticeID(ctx context.Context, noticeID string) (*Opportunity, error) {
	opp := &Opportunity{}
	err := db.pool.QueryRow(ctx, `
		SELECT
			notice_id, solicitation_number, title, description,
			type, base_type, posted_date, response_deadline,
			set_aside_code, naics_code, data_source
		FROM opportunities
		WHERE notice_id = $1
	`, noticeID).Scan(
		&opp.NoticeID, &opp.SolicitationNumber, &opp.Title, &opp.Description,
		&opp.Type, &opp.BaseType, &opp.PostedDate, &opp.ResponseDeadline,
		&opp.SetAsideCode, &opp.NAICSCode, &opp.DataSource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get opportunity %s: %w", noticeID, err)
	}
	return opp, nil
}

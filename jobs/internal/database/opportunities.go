package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/jobs/internal/reconcile"
	"github.com/jackc/pgx/v5"
)

const upsertBatchSize = 500

func (db *DB) UpsertOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (inserted, updated int, err error) {
	for i := 0; i < len(opps); i += upsertBatchSize {
		end := i + upsertBatchSize
		if end > len(opps) {
			end = len(opps)
		}
		ins, upd, batchErr := db.upsertOpportunitiesBatch(ctx, runID, snapshotDate, opps[i:end])
		if batchErr != nil {
			return inserted, updated, fmt.Errorf("upsert batch %d-%d: %w", i, end, batchErr)
		}
		inserted += ins
		updated += upd
	}
	return inserted, updated, nil
}

func (db *DB) upsertOpportunitiesBatch(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (inserted, updated int, err error) {
	batch := &pgx.Batch{}

	for _, o := range opps {
		if o.NoticeID == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO opportunities (
				notice_id, solicitation_number, title, description, type, base_type, organization_type,
				posted_date, response_deadline, archive_date, archive_type, active,
				set_aside_code, set_aside_description, naics_code, classification_code,
				department, sub_tier, office, cgac, fpds_code, aac_code,
				pop_street_address, pop_city, pop_state, pop_zip, pop_country,
				office_city, office_state, office_zip, office_country,
				award_number, award_date, award_amount, awardee,
				primary_contact_title, primary_contact_fullname, primary_contact_email, primary_contact_phone, primary_contact_fax,
				secondary_contact_title, secondary_contact_fullname, secondary_contact_email, secondary_contact_phone, secondary_contact_fax,
				ui_link,
				data_sources, last_csv_run_id, last_seen_csv
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12,
				$13, $14, $15, $16,
				$17, $18, $19, $20, $21, $22,
				$23, $24, $25, $26, $27,
				$28, $29, $30, $31,
				$32, $33, $34, $35,
				$36, $37, $38, $39, $40,
				$41, $42, $43, $44, $45,
				$46,
				'csv', $47, $48
			)
			ON CONFLICT (notice_id) DO UPDATE SET
				solicitation_number = EXCLUDED.solicitation_number,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				type = EXCLUDED.type,
				base_type = EXCLUDED.base_type,
				organization_type = EXCLUDED.organization_type,
				posted_date = EXCLUDED.posted_date,
				response_deadline = EXCLUDED.response_deadline,
				archive_date = EXCLUDED.archive_date,
				archive_type = EXCLUDED.archive_type,
				active = EXCLUDED.active,
				set_aside_code = EXCLUDED.set_aside_code,
				set_aside_description = EXCLUDED.set_aside_description,
				naics_code = EXCLUDED.naics_code,
				classification_code = EXCLUDED.classification_code,
				department = EXCLUDED.department,
				sub_tier = EXCLUDED.sub_tier,
				office = EXCLUDED.office,
				cgac = EXCLUDED.cgac,
				fpds_code = EXCLUDED.fpds_code,
				aac_code = EXCLUDED.aac_code,
				pop_street_address = EXCLUDED.pop_street_address,
				pop_city = EXCLUDED.pop_city,
				pop_state = EXCLUDED.pop_state,
				pop_zip = EXCLUDED.pop_zip,
				pop_country = EXCLUDED.pop_country,
				office_city = EXCLUDED.office_city,
				office_state = EXCLUDED.office_state,
				office_zip = EXCLUDED.office_zip,
				office_country = EXCLUDED.office_country,
				award_number = EXCLUDED.award_number,
				award_date = EXCLUDED.award_date,
				award_amount = EXCLUDED.award_amount,
				awardee = EXCLUDED.awardee,
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
				ui_link = EXCLUDED.ui_link,
				data_sources = CASE
					WHEN opportunities.data_sources LIKE '%api%' THEN 'csv+api'
					ELSE 'csv'
				END,
				last_csv_run_id = EXCLUDED.last_csv_run_id,
				last_seen_csv = EXCLUDED.last_seen_csv
		`,
			o.NoticeID, nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
			nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
			o.PostedDate, o.ResponseDeadline, nilIfEmpty(o.ArchiveDate), nilIfEmpty(o.ArchiveType), o.Active,
			nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
			nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
			nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
			nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
			nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
			nilIfEmpty(o.AwardNumber), nilIfEmpty(o.AwardDate), o.AwardAmount, nilIfEmpty(o.Awardee),
			nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
			nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
			nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
			nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
			nilIfEmpty(o.UILink),
			runID, snapshotDate,
		)
	}

	if batch.Len() == 0 {
		return 0, 0, nil
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < batch.Len(); i++ {
		tag, err := results.Exec()
		if err != nil {
			return inserted, updated, fmt.Errorf("exec upsert %d: %w", i, err)
		}
		// INSERT returns RowsAffected=1 for both insert and update via ON CONFLICT.
		// xmax=0 means insert, xmax!=0 means update — but we can't read that from tag.
		// Use a heuristic: if the row existed, updated_at trigger fires.
		// For stats, we just count total and derive later.
		if tag.RowsAffected() > 0 {
			inserted++
		}
	}

	return inserted, 0, nil
}

func (db *DB) MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE opportunities o
		SET active = false
		FROM pipeline.snap_disappearances d
		WHERE d.notice_id = o.notice_id
		  AND d.run_id = $1
		  AND d.resolution IS NULL
	`, runID)
	if err != nil {
		return 0, fmt.Errorf("mark disappeared inactive: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// GetSnapCSVRawData loads raw_data for all rows of a given run from snap_csv.
func (db *DB) GetSnapCSVRawData(ctx context.Context, runID uuid.UUID) ([]map[string]string, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT raw_data FROM pipeline.snap_csv WHERE run_id = $1
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("query snap_csv raw_data: %w", err)
	}
	defer rows.Close()

	var result []map[string]string
	for rows.Next() {
		var rawJSON json.RawMessage
		if err := rows.Scan(&rawJSON); err != nil {
			return nil, fmt.Errorf("scan raw_data: %w", err)
		}
		var raw map[string]string
		if err := json.Unmarshal(rawJSON, &raw); err != nil {
			return nil, fmt.Errorf("unmarshal raw_data: %w", err)
		}
		result = append(result, raw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snap_csv rows: %w", err)
	}
	return result, nil
}

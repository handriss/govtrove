package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/reconcile"
	"github.com/jackc/pgx/v5"
)

const upsertBatchSize = 500

type existingRow struct {
	ContentHash string
	Version     int
}

func (db *DB) UpsertOpportunities(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (affected int, err error) {
	for i := 0; i < len(opps); i += upsertBatchSize {
		end := i + upsertBatchSize
		if end > len(opps) {
			end = len(opps)
		}
		batchAffected, batchFailed, batchErr := db.upsertOpportunitiesBatch(ctx, runID, snapshotDate, opps[i:end])
		if batchErr != nil {
			return affected, fmt.Errorf("upsert batch %d-%d: %w", i, end, batchErr)
		}
		if batchFailed > 0 {
			slog.Warn("batch had row failures", "batch_start", i, "failed", batchFailed)
		}
		affected += batchAffected
	}
	return affected, nil
}

func (db *DB) upsertOpportunitiesBatch(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, opps []reconcile.Opportunity) (affected, failed int, err error) {
	// Build list of valid opportunities with their hashes
	type oppWithHash struct {
		opp  reconcile.Opportunity
		hash string
	}
	var valid []oppWithHash
	var noticeIDs []string
	for _, o := range opps {
		if o.NoticeID == "" {
			continue
		}
		valid = append(valid, oppWithHash{opp: o, hash: o.ContentHash()})
		noticeIDs = append(noticeIDs, o.NoticeID)
	}
	if len(valid) == 0 {
		return 0, 0, nil
	}

	// Query existing latest rows for these notice_ids
	existing := make(map[string]existingRow)
	rows, err := db.pool.Query(ctx, `
		SELECT notice_id, COALESCE(content_hash, ''), version
		FROM opportunities
		WHERE notice_id = ANY($1) AND is_latest = true
	`, noticeIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("query existing rows: %w", err)
	}
	for rows.Next() {
		var nid, hash string
		var ver int
		if err := rows.Scan(&nid, &hash, &ver); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan existing row: %w", err)
		}
		existing[nid] = existingRow{ContentHash: hash, Version: ver}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterate existing rows: %w", err)
	}

	batch := &pgx.Batch{}
	var batchNoticeIDs []string

	for _, v := range valid {
		o := v.opp
		hash := v.hash

		ex, exists := existing[o.NoticeID]

		if !exists {
			// New notice_id → INSERT version 1
			batch.Queue(csvInsertSQL,
				o.NoticeID, nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
				nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
				o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,
				nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
				nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
				nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
				nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
				nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
				nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),
				nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
				nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
				nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
				nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
				nilIfEmpty(o.UILink),
				1, true, hash,
				runID, snapshotDate,
			)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else if ex.ContentHash == hash {
			// Same hash → metadata-only update (includes active flag which is excluded from hash)
			batch.Queue(`
				UPDATE opportunities SET last_csv_run_id = $1, last_seen_csv = $2, active = $3,
					data_sources = CASE WHEN data_sources LIKE '%api%' THEN 'csv+api' ELSE 'csv' END
				WHERE notice_id = $4 AND is_latest = true
			`, runID, snapshotDate, o.Active, o.NoticeID)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else if ex.ContentHash == "" {
			// Pre-migration row with no hash yet — backfill hash + update content, no new version
			batch.Queue(`
				UPDATE opportunities SET
					solicitation_number = $1, title = $2, description = $3, type = $4, base_type = $5, organization_type = $6,
					posted_date = $7, response_deadline = $8, archive_date = $9, archive_type = $10, active = $11,
					set_aside_code = $12, set_aside_description = $13, naics_code = $14, classification_code = $15,
					department = $16, sub_tier = $17, office = $18, cgac = $19, fpds_code = $20, aac_code = $21,
					pop_street_address = $22, pop_city = $23, pop_state = $24, pop_zip = $25, pop_country = $26,
					office_city = $27, office_state = $28, office_zip = $29, office_country = $30,
					award_number = $31, award_date = $32, award_amount = $33, awardee = $34,
					primary_contact_title = $35, primary_contact_fullname = $36, primary_contact_email = $37,
					primary_contact_phone = $38, primary_contact_fax = $39,
					secondary_contact_title = $40, secondary_contact_fullname = $41, secondary_contact_email = $42,
					secondary_contact_phone = $43, secondary_contact_fax = $44,
					ui_link = $45, content_hash = $46,
					data_sources = CASE WHEN data_sources LIKE '%api%' THEN 'csv+api' ELSE 'csv' END,
					last_csv_run_id = $47, last_seen_csv = $48
				WHERE notice_id = $49 AND is_latest = true
			`,
				nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
				nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
				o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,
				nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
				nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
				nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
				nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
				nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
				nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),
				nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
				nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
				nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
				nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
				nilIfEmpty(o.UILink), hash,
				runID, snapshotDate, o.NoticeID,
			)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else {
			// Different hash → mark old as not-latest, insert new version
			newVersion := ex.Version + 1
			batch.Queue(`
				UPDATE opportunities SET is_latest = false
				WHERE notice_id = $1 AND is_latest = true
			`, o.NoticeID)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID+":demote")

			batch.Queue(csvInsertSQL,
				o.NoticeID, nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description),
				nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),
				o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,
				nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription), nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),
				nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),
				nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode),
				nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState), nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),
				nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry),
				nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),
				nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname), nilIfEmpty(o.PrimaryContactEmail),
				nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax),
				nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname), nilIfEmpty(o.SecondaryContactEmail),
				nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax),
				nilIfEmpty(o.UILink),
				newVersion, true, hash,
				runID, snapshotDate,
			)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		}
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < batch.Len(); i++ {
		tag, err := results.Exec()
		if err != nil {
			noticeID := ""
			if i < len(batchNoticeIDs) {
				noticeID = batchNoticeIDs[i]
			}
			slog.Warn("upsert row failed", "notice_id", noticeID, "error", err)
			failed++
			continue
		}
		if tag.RowsAffected() > 0 {
			affected++
		}
	}

	return affected, failed, nil
}

const csvInsertSQL = `
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
		version, is_latest, content_hash,
		data_sources, last_csv_run_id, last_seen_csv,
		full_parent_path_name
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
		$47, $48, $49,
		'csv', $50, $51,
		CASE
			WHEN $19::text IS NOT NULL THEN CONCAT_WS('.', $17::text, $18::text, $19::text)
			WHEN $18::text IS NOT NULL THEN CONCAT_WS('.', $17::text, $18::text)
			WHEN $17::text IS NOT NULL THEN $17::text
			ELSE NULL
		END
	)`

func (db *DB) MarkDisappearedInactive(ctx context.Context, runID uuid.UUID) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE opportunities o
		SET active = false
		FROM pipeline.snap_disappearances d
		WHERE d.notice_id = o.notice_id
		  AND d.run_id = $1
		  AND d.resolution IS NULL
		  AND o.is_latest = true
	`, runID)
	if err != nil {
		return 0, fmt.Errorf("mark disappeared inactive: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ResolveExpectedDisappearances marks disappearances as "archived" when the
// notice_id appears in the archived CSV run. Returns the count resolved.
func (db *DB) ResolveExpectedDisappearances(ctx context.Context, activeRunID, archivedRunID uuid.UUID, snapshotDate time.Time) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE pipeline.snap_disappearances d
		SET resolution = 'archived',
		    resolution_date = $3,
		    resolution_source = 'archived_csv'
		FROM pipeline.snap_csv a
		WHERE a.notice_id = d.notice_id
		  AND a.run_id = $2
		  AND d.run_id = $1
		  AND d.resolution IS NULL
	`, activeRunID, archivedRunID, snapshotDate)
	if err != nil {
		return 0, fmt.Errorf("resolve expected disappearances: %w", err)
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

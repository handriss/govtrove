package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

type UnchangedRecord struct {
	NoticeID    string
	Active      bool
	DataSources string
}

func (db *DB) GetExistingOpportunityHashes(ctx context.Context) (map[string]string, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT notice_id, COALESCE(content_hash, '')
		FROM opportunities
		WHERE is_latest = true
	`)
	if err != nil {
		return nil, fmt.Errorf("query opportunity hashes: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var nid, hash string
		if err := rows.Scan(&nid, &hash); err != nil {
			return nil, fmt.Errorf("scan opportunity hash: %w", err)
		}
		result[nid] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate opportunity hashes: %w", err)
	}
	return result, nil
}

func (db *DB) BulkTouchUnchanged(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []UnchangedRecord) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	const batchSize = 10000
	total := 0
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		noticeIDs := make([]string, len(batch))
		actives := make([]bool, len(batch))
		sources := make([]string, len(batch))
		for j, r := range batch {
			noticeIDs[j] = r.NoticeID
			actives[j] = r.Active
			sources[j] = r.DataSources
		}

		tag, err := db.pool.Exec(ctx, `
			UPDATE opportunities o
			SET last_csv_run_id = $1, last_seen_csv = $2, active = u.active, data_sources = u.data_sources
			FROM unnest($3::text[], $4::boolean[], $5::text[]) AS u(notice_id, active, data_sources)
			WHERE o.notice_id = u.notice_id AND o.is_latest = true
		`, runID, snapshotDate, noticeIDs, actives, sources)
		if err != nil {
			return total, fmt.Errorf("bulk touch unchanged batch %d-%d: %w", i, end, err)
		}
		total += int(tag.RowsAffected())
	}
	return total, nil
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
		params := oppContentParams(o)

		ex, exists := existing[o.NoticeID]

		if !exists {
			// New notice_id → INSERT version 1
			args := append(params, 1, true, hash, o.DataSources, runID, snapshotDate)
			batch.Queue(insertSQL, args...)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else if ex.ContentHash == hash {
			// Same hash → metadata-only update (includes active flag which is excluded from hash)
			batch.Queue(`
				UPDATE opportunities SET last_csv_run_id = $1, last_seen_csv = $2, active = $3,
					data_sources = $4
				WHERE notice_id = $5 AND is_latest = true
			`, runID, snapshotDate, o.Active, o.DataSources, o.NoticeID)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else if ex.ContentHash == "" {
			// Pre-migration row with no hash yet — backfill hash + update content, no new version
			args := append(params[1:], hash, o.DataSources, runID, snapshotDate, o.NoticeID) // skip notice_id ($1 of insert)
			batch.Queue(backfillSQL, args...)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID)
		} else {
			// Different hash → mark old as not-latest, insert new version
			newVersion := ex.Version + 1
			batch.Queue(`
				UPDATE opportunities SET is_latest = false
				WHERE notice_id = $1 AND is_latest = true
			`, o.NoticeID)
			batchNoticeIDs = append(batchNoticeIDs, o.NoticeID+":demote")

			args := append(params, newVersion, true, hash, o.DataSources, runID, snapshotDate)
			batch.Queue(insertSQL, args...)
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

// oppContentParams builds the ordered content parameters for an Opportunity.
// Used by both insertSQL and backfillSQL to avoid duplicating the param list.
func oppContentParams(o reconcile.Opportunity) []interface{} {
	return []interface{}{
		o.NoticeID, nilIfEmpty(o.SolicitationNumber), nilIfEmpty(o.Title), nilIfEmpty(o.Description), // 1-4
		nilIfEmpty(o.Type), nilIfEmpty(o.BaseType), nilIfEmpty(o.OrganizationType),                   // 5-7
		o.PostedDate, o.ResponseDeadline, o.ArchiveDate, nilIfEmpty(o.ArchiveType), o.Active,         // 8-12
		nilIfEmpty(o.SetAsideCode), nilIfEmpty(o.SetAsideDescription),                                // 13-14
		nilIfEmpty(o.NAICSCode), nilIfEmpty(o.ClassificationCode),                                    // 15-16
		nilIfEmpty(o.Department), nilIfEmpty(o.SubTier), nilIfEmpty(o.Office),                         // 17-19
		nilIfEmpty(o.CGAC), nilIfEmpty(o.FPDSCode), nilIfEmpty(o.AACCode), nilIfEmpty(o.MiddleTier),  // 20-23
		nilIfEmpty(o.PopStreetAddress), nilIfEmpty(o.PopCity), nilIfEmpty(o.PopState),                 // 24-26
		nilIfEmpty(o.PopZip), nilIfEmpty(o.PopCountry),                                               // 27-28
		nilIfEmpty(o.PopCityCode), nilIfEmpty(o.PopStateCode), nilIfEmpty(o.PopCountryCode),          // 29-31
		nilIfEmpty(o.OfficeCity), nilIfEmpty(o.OfficeState), nilIfEmpty(o.OfficeZip), nilIfEmpty(o.OfficeCountry), // 32-35
		nilIfEmpty(o.AwardNumber), o.AwardDate, o.AwardAmount, nilIfEmpty(o.Awardee),                 // 36-39
		nilIfEmpty(o.AwardeeName), nilIfEmpty(o.AwardeeUeiSAM), nilIfEmpty(o.AwardeeStreetAddress),   // 40-42
		nilIfEmpty(o.AwardeeCity), nilIfEmpty(o.AwardeeCityCode),                                     // 43-44
		nilIfEmpty(o.AwardeeState), nilIfEmpty(o.AwardeeStateCode),                                   // 45-46
		nilIfEmpty(o.AwardeeCountry), nilIfEmpty(o.AwardeeCountryCode), nilIfEmpty(o.AwardeeZip),     // 47-49
		nilIfEmpty(o.PrimaryContactTitle), nilIfEmpty(o.PrimaryContactFullname),                       // 50-51
		nilIfEmpty(o.PrimaryContactEmail), nilIfEmpty(o.PrimaryContactPhone), nilIfEmpty(o.PrimaryContactFax), // 52-54
		nilIfEmpty(o.SecondaryContactTitle), nilIfEmpty(o.SecondaryContactFullname),                   // 55-56
		nilIfEmpty(o.SecondaryContactEmail), nilIfEmpty(o.SecondaryContactPhone), nilIfEmpty(o.SecondaryContactFax), // 57-59
		nilIfEmpty(o.UILink), nilIfEmpty(o.AdditionalInfoLink), nilIfEmpty(o.DescriptionURL),          // 60-62
		resourceLinksJSON(o.ResourceLinks),                                                            // 63
		computeFullParentPathName(o), nilIfEmpty(o.FullParentPathCode),                                // 64-65
	}
}

func resourceLinksJSON(links []string) interface{} {
	if len(links) == 0 {
		return nil
	}
	b, _ := json.Marshal(links)
	return string(b)
}

func computeFullParentPathName(o reconcile.Opportunity) *string {
	if o.FullParentPathName != "" {
		return &o.FullParentPathName
	}
	var parts []string
	if o.Department != "" {
		parts = append(parts, o.Department)
	}
	if o.SubTier != "" {
		parts = append(parts, o.SubTier)
	}
	if o.Office != "" {
		parts = append(parts, o.Office)
	}
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, ".")
	return &s
}

// insertSQL: 65 content params + version($66), is_latest($67), content_hash($68),
// data_sources($69), last_csv_run_id($70), last_seen_csv($71)
const insertSQL = `
	INSERT INTO opportunities (
		notice_id, solicitation_number, title, description, type, base_type, organization_type,
		posted_date, response_deadline, archive_date, archive_type, active,
		set_aside_code, set_aside_description, naics_code, classification_code,
		department, sub_tier, office, cgac, fpds_code, aac_code, middle_tier,
		pop_street_address, pop_city, pop_state, pop_zip, pop_country,
		pop_city_code, pop_state_code, pop_country_code,
		office_city, office_state, office_zip, office_country,
		award_number, award_date, award_amount, awardee,
		awardee_name, awardee_uei, awardee_street_address,
		awardee_city, awardee_city_code, awardee_state, awardee_state_code,
		awardee_country, awardee_country_code, awardee_zip,
		primary_contact_title, primary_contact_fullname, primary_contact_email, primary_contact_phone, primary_contact_fax,
		secondary_contact_title, secondary_contact_fullname, secondary_contact_email, secondary_contact_phone, secondary_contact_fax,
		ui_link, additional_info_link, description_url, resource_links,
		full_parent_path_name, full_parent_path_code,
		version, is_latest, content_hash,
		data_sources, last_csv_run_id, last_seen_csv
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12,
		$13, $14, $15, $16,
		$17, $18, $19, $20, $21, $22, $23,
		$24, $25, $26, $27, $28,
		$29, $30, $31,
		$32, $33, $34, $35,
		$36, $37, $38, $39,
		$40, $41, $42,
		$43, $44, $45, $46,
		$47, $48, $49,
		$50, $51, $52, $53, $54,
		$55, $56, $57, $58, $59,
		$60, $61, $62, $63,
		$64, $65,
		$66, $67, $68,
		$69, $70, $71
	)`

// backfillSQL: same content params (minus notice_id) + content_hash, data_sources, run tracking, WHERE.
// Params: content[2:65] as $1-$64, content_hash=$65, data_sources=$66, run_id=$67, seen=$68, notice_id=$69
const backfillSQL = `
	UPDATE opportunities SET
		solicitation_number = $1, title = $2, description = $3, type = $4, base_type = $5, organization_type = $6,
		posted_date = $7, response_deadline = $8, archive_date = $9, archive_type = $10, active = $11,
		set_aside_code = $12, set_aside_description = $13, naics_code = $14, classification_code = $15,
		department = $16, sub_tier = $17, office = $18, cgac = $19, fpds_code = $20, aac_code = $21, middle_tier = $22,
		pop_street_address = $23, pop_city = $24, pop_state = $25, pop_zip = $26, pop_country = $27,
		pop_city_code = $28, pop_state_code = $29, pop_country_code = $30,
		office_city = $31, office_state = $32, office_zip = $33, office_country = $34,
		award_number = $35, award_date = $36, award_amount = $37, awardee = $38,
		awardee_name = $39, awardee_uei = $40, awardee_street_address = $41,
		awardee_city = $42, awardee_city_code = $43, awardee_state = $44, awardee_state_code = $45,
		awardee_country = $46, awardee_country_code = $47, awardee_zip = $48,
		primary_contact_title = $49, primary_contact_fullname = $50, primary_contact_email = $51, primary_contact_phone = $52, primary_contact_fax = $53,
		secondary_contact_title = $54, secondary_contact_fullname = $55, secondary_contact_email = $56, secondary_contact_phone = $57, secondary_contact_fax = $58,
		ui_link = $59, additional_info_link = $60, description_url = $61, resource_links = $62,
		full_parent_path_name = $63, full_parent_path_code = $64,
		content_hash = $65,
		data_sources = $66,
		last_csv_run_id = $67, last_seen_csv = $68
	WHERE notice_id = $69 AND is_latest = true
`

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

func (db *DB) DeactivateExpiredOpportunities(ctx context.Context) (expired int, stale int, err error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE opportunities SET active = false
		WHERE active = true AND is_latest = true AND archive_date < CURRENT_DATE
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("deactivate expired: %w", err)
	}
	expired = int(tag.RowsAffected())

	tag, err = db.pool.Exec(ctx, `
		UPDATE opportunities SET active = false
		WHERE active = true AND is_latest = true AND archive_date IS NULL
		  AND COALESCE(response_deadline, posted_date) < NOW() - INTERVAL '90 days'
	`)
	if err != nil {
		return expired, 0, fmt.Errorf("deactivate stale: %w", err)
	}
	stale = int(tag.RowsAffected())

	return expired, stale, nil
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

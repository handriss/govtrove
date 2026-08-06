package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SnapCSVRow struct {
	NoticeID           string
	SolicitationNumber string
	Title              string
	Type               string
	BaseType           string
	PostedDate         *time.Time
	ResponseDeadline   *time.Time
	ArchiveDate        string
	ArchiveType        string
	SetAsideCode       string
	NAICSCode          string
	ClassificationCode string
	Active             bool

	Department string
	SubTier    string
	Office     string
	CGAC       string
	FPDSCode   string
	AACCode    string

	AwardNumber string
	AwardDate   string
	AwardAmount *float64

	RawData     map[string]string
	ContentHash string
}

func ComputeContentHash(row map[string]string) string {
	data, _ := json.Marshal(row)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (db *DB) BulkInsertSnapCSV(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, downloadID int64, rows []SnapCSVRow) (int64, error) {
	columns := []string{
		"notice_id", "solicitation_number", "title", "type", "base_type",
		"posted_date", "response_deadline", "archive_date", "archive_type",
		"set_aside_code", "naics_code", "classification_code", "active",
		"department", "sub_tier", "office", "cgac", "fpds_code", "aac_code",
		"award_number", "award_date", "award_amount",
		"raw_data", "content_hash",
		"run_id", "snapshot_date", "download_id",
	}

	copyCount, err := db.pool.CopyFrom(
		ctx,
		pgx.Identifier{"pipeline", "snap_csv"},
		columns,
		&snapCSVCopySource{rows: rows, runID: runID, snapshotDate: snapshotDate, downloadID: downloadID},
	)
	if err != nil {
		return 0, fmt.Errorf("bulk insert snap_csv failed: %w", err)
	}
	return copyCount, nil
}

type snapCSVCopySource struct {
	rows         []SnapCSVRow
	runID        uuid.UUID
	snapshotDate time.Time
	downloadID   int64
	idx          int
}

func (s *snapCSVCopySource) Next() bool {
	s.idx++
	return s.idx <= len(s.rows)
}

func (s *snapCSVCopySource) Values() ([]interface{}, error) {
	row := s.rows[s.idx-1]
	rawJSON, _ := json.Marshal(row.RawData)

	return []interface{}{
		row.NoticeID, nilIfEmpty(row.SolicitationNumber), nilIfEmpty(row.Title),
		nilIfEmpty(row.Type), nilIfEmpty(row.BaseType),
		row.PostedDate, row.ResponseDeadline,
		nilIfEmpty(row.ArchiveDate), nilIfEmpty(row.ArchiveType),
		nilIfEmpty(row.SetAsideCode), nilIfEmpty(row.NAICSCode), nilIfEmpty(row.ClassificationCode),
		row.Active,
		nilIfEmpty(row.Department), nilIfEmpty(row.SubTier), nilIfEmpty(row.Office),
		nilIfEmpty(row.CGAC), nilIfEmpty(row.FPDSCode), nilIfEmpty(row.AACCode),
		nilIfEmpty(row.AwardNumber), nilIfEmpty(row.AwardDate), row.AwardAmount,
		rawJSON, row.ContentHash,
		s.runID, s.snapshotDate, s.downloadID,
	}, nil
}

func (s *snapCSVCopySource) Err() error { return nil }

// PreviousRunRecord holds the data needed for in-memory change detection.
type PreviousRunRecord struct {
	ContentHash        string
	SolicitationNumber string
	Type               string
	ArchiveType        string
	ArchiveDate        string
	SnapshotDate       time.Time
}

func (db *DB) GetPreviousRunHashes(ctx context.Context, runID uuid.UUID) (map[string]PreviousRunRecord, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT notice_id, content_hash, solicitation_number, type, archive_type, archive_date, snapshot_date
		FROM pipeline.snap_csv
		WHERE run_id = $1
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("query previous run hashes: %w", err)
	}
	defer rows.Close()

	result := make(map[string]PreviousRunRecord, 80000)
	for rows.Next() {
		var noticeID string
		var rec PreviousRunRecord
		var solNum, typ, archType, archDate *string
		if err := rows.Scan(&noticeID, &rec.ContentHash, &solNum, &typ, &archType, &archDate, &rec.SnapshotDate); err != nil {
			return nil, fmt.Errorf("scan previous run record: %w", err)
		}
		if solNum != nil {
			rec.SolicitationNumber = *solNum
		}
		if typ != nil {
			rec.Type = *typ
		}
		if archType != nil {
			rec.ArchiveType = *archType
		}
		if archDate != nil {
			rec.ArchiveDate = *archDate
		}
		result[noticeID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate previous run records: %w", err)
	}
	return result, nil
}

// DisappearedRecord holds metadata for a record that vanished from the active CSV.
type DisappearedRecord struct {
	NoticeID           string
	SolicitationNumber string
	Type               string
	ArchiveType        string
	ArchiveDate        string
	LastSeenDate       time.Time
}

func (db *DB) InsertDisappearances(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, records []DisappearedRecord, logger *slog.Logger) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, r := range records {
		batch.Queue(`
			INSERT INTO pipeline.snap_disappearances (run_id, notice_id, solicitation_number, last_seen_date, disappeared_date, last_type, last_archive_type, last_archive_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, runID, r.NoticeID, nilIfEmpty(r.SolicitationNumber), r.LastSeenDate, snapshotDate, nilIfEmpty(r.Type), nilIfEmpty(r.ArchiveType), nilIfEmpty(r.ArchiveDate))
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			logger.Warn("failed to insert snap_disappearance", "error", err)
		}
	}

	return len(records), nil
}

func (db *DB) DetectReappearances(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE pipeline.snap_disappearances d
		SET resolution = 'glitch',
		    resolution_date = $2,
		    resolution_source = 'active_csv_reappeared',
		    reappeared_date = $2
		FROM pipeline.snap_csv c
		WHERE d.notice_id = c.notice_id
		  AND c.run_id = $1
		  AND d.resolution IS NULL
	`, currentRunID, snapshotDate)
	if err != nil {
		return 0, fmt.Errorf("detect reappearances: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

type DataQualityEntry struct {
	NoticeID     string
	SnapshotDate time.Time
	Source       string
	IssueType    string
	FieldName    string
	FieldValue   string
}

func (db *DB) InsertDataQualityIssues(ctx context.Context, runID uuid.UUID, entries []DataQualityEntry) {
	if len(entries) == 0 {
		return
	}

	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(`
			INSERT INTO pipeline.snap_data_quality (run_id, notice_id, snapshot_date, source, issue_type, field_name, field_value)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, runID, e.NoticeID, e.SnapshotDate, e.Source, e.IssueType, nilIfEmpty(e.FieldName), nilIfEmpty(e.FieldValue))
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			slog.Warn("failed to insert data quality issue", "error", err, "notice_id", entries[i].NoticeID)
		}
	}
}

type ReconcileDQEntry struct {
	NoticeID     string
	SnapshotDate time.Time
	IssueType    string
	FieldName    string
	CSVValue     string
	APIValue     string
}

func (db *DB) InsertReconcileDQIssues(ctx context.Context, csvRunID, apiRunID uuid.UUID, entries []ReconcileDQEntry) {
	if len(entries) == 0 {
		return
	}

	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(`
			INSERT INTO pipeline.snap_reconcile_dq (csv_run_id, api_run_id, notice_id, snapshot_date, issue_type, field_name, csv_value, api_value)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, nilUUID(csvRunID), nilUUID(apiRunID), e.NoticeID, e.SnapshotDate, e.IssueType, e.FieldName, nilIfEmpty(e.CSVValue), nilIfEmpty(e.APIValue))
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			slog.Warn("failed to insert reconcile DQ issue", "error", err, "notice_id", entries[i].NoticeID)
		}
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nilUUID maps a zero UUID to SQL NULL so the snap_reconcile_dq FK to
// ingestion_runs stays satisfied when a reconcile ran with only one source
// present (API-only during a CSV outage, or the reverse).
func nilUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

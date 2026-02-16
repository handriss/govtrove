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

type SnapChange struct {
	NoticeID           string
	SolicitationNumber string
	FieldName          string
	OldValue           string
	NewValue           string
	ChangeType         string
}

type CSVDownloadEntry struct {
	RunID         uuid.UUID
	URL           string
	Status        string
	ETag          string
	LastModified  string
	FileSizeBytes int64
	RecordCount   int
	ErrorMessage  string
}

func ComputeContentHash(row map[string]string) string {
	data, _ := json.Marshal(row)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (db *DB) CreateCSVDownloadEntry(ctx context.Context, e *CSVDownloadEntry) (int64, error) {
	var id int64
	err := db.pool.QueryRow(ctx, `
		INSERT INTO csv_download_log (run_id, url, status, etag, last_modified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, e.RunID, e.URL, e.Status, nilIfEmpty(e.ETag), nilIfEmpty(e.LastModified)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create csv_download_log entry: %w", err)
	}
	return id, nil
}

func (db *DB) CompleteCSVDownloadEntry(ctx context.Context, id int64, recordCount int, fileSizeBytes int64) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE csv_download_log
		SET status = 'completed', record_count = $2, file_size_bytes = $3
		WHERE id = $1
	`, id, recordCount, fileSizeBytes)
	return err
}

func (db *DB) FailCSVDownloadEntry(ctx context.Context, id int64, errMsg string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE csv_download_log
		SET status = 'failed', error_message = $2
		WHERE id = $1
	`, id, errMsg)
	return err
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
		pgx.Identifier{"snap_csv"},
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

func (db *DB) DetectChanges(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (newCount, changedCount int, err error) {
	// Find new records (in current but not in previous)
	newRows, err := db.pool.Query(ctx, `
		SELECT c.notice_id, c.solicitation_number
		FROM snap_csv c
		LEFT JOIN snap_csv p ON c.notice_id = p.notice_id AND p.run_id = $2
		WHERE c.run_id = $1 AND p.notice_id IS NULL
	`, currentRunID, previousRunID)
	if err != nil {
		return 0, 0, fmt.Errorf("detect new records: %w", err)
	}
	defer newRows.Close()

	batch := &pgx.Batch{}
	for newRows.Next() {
		var noticeID string
		var solNum *string
		if err := newRows.Scan(&noticeID, &solNum); err != nil {
			return 0, 0, fmt.Errorf("scan new record: %w", err)
		}
		newCount++
		batch.Queue(`
			INSERT INTO snap_changes (run_id, notice_id, solicitation_number, source, field_name, new_value, detected_date, change_type)
			VALUES ($1, $2, $3, 'active_csv', '_record', 'new', $4, 'new')
		`, currentRunID, noticeID, solNum, snapshotDate)
	}
	if err := newRows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterate new records: %w", err)
	}

	// Find changed records (hash mismatch)
	changedRows, err := db.pool.Query(ctx, `
		SELECT c.notice_id, c.solicitation_number, c.raw_data, p.raw_data
		FROM snap_csv c
		JOIN snap_csv p ON c.notice_id = p.notice_id AND p.run_id = $2
		WHERE c.run_id = $1 AND c.content_hash != p.content_hash
	`, currentRunID, previousRunID)
	if err != nil {
		return newCount, 0, fmt.Errorf("detect changed records: %w", err)
	}
	defer changedRows.Close()

	previousDate := snapshotDate // approximate — use previous run's snapshot_date ideally
	// Get previous run's start time for previous_date
	var prevDate time.Time
	pErr := db.pool.QueryRow(ctx, `SELECT started_at FROM ingestion_runs WHERE run_id = $1`, previousRunID).Scan(&prevDate)
	if pErr == nil {
		previousDate = prevDate
	}

	for changedRows.Next() {
		var noticeID string
		var solNum *string
		var currentRaw, previousRaw json.RawMessage
		if err := changedRows.Scan(&noticeID, &solNum, &currentRaw, &previousRaw); err != nil {
			return newCount, changedCount, fmt.Errorf("scan changed record: %w", err)
		}
		changedCount++

		var curr, prev map[string]string
		json.Unmarshal(currentRaw, &curr)
		json.Unmarshal(previousRaw, &prev)

		allKeys := make(map[string]bool)
		for k := range curr {
			allKeys[k] = true
		}
		for k := range prev {
			allKeys[k] = true
		}

		for field := range allKeys {
			oldVal := prev[field]
			newVal := curr[field]
			if oldVal != newVal {
				batch.Queue(`
					INSERT INTO snap_changes (run_id, notice_id, solicitation_number, source, field_name, old_value, new_value, detected_date, previous_date, change_type)
					VALUES ($1, $2, $3, 'active_csv', $4, $5, $6, $7, $8, 'modified')
				`, currentRunID, noticeID, solNum, field, nilIfEmpty(oldVal), nilIfEmpty(newVal), snapshotDate, previousDate)
			}
		}
	}
	if err := changedRows.Err(); err != nil {
		return newCount, changedCount, fmt.Errorf("iterate changed records: %w", err)
	}

	if batch.Len() > 0 {
		results := db.pool.SendBatch(ctx, batch)
		defer results.Close()
		for i := 0; i < batch.Len(); i++ {
			if _, err := results.Exec(); err != nil {
				logger.Warn("failed to insert snap_change", "error", err)
			}
		}
	}

	return newCount, changedCount, nil
}

func (db *DB) DetectDisappearances(ctx context.Context, currentRunID, previousRunID uuid.UUID, snapshotDate time.Time, logger *slog.Logger) (int, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT p.notice_id, p.solicitation_number, p.type, p.archive_type, p.archive_date, p.snapshot_date
		FROM snap_csv p
		LEFT JOIN snap_csv c ON p.notice_id = c.notice_id AND c.run_id = $1
		WHERE p.run_id = $2 AND c.notice_id IS NULL
	`, currentRunID, previousRunID)
	if err != nil {
		return 0, fmt.Errorf("detect disappearances: %w", err)
	}
	defer rows.Close()

	batch := &pgx.Batch{}
	count := 0
	for rows.Next() {
		var noticeID string
		var solNum, typ, archType, archDate *string
		var lastSeen time.Time
		if err := rows.Scan(&noticeID, &solNum, &typ, &archType, &archDate, &lastSeen); err != nil {
			return count, fmt.Errorf("scan disappeared record: %w", err)
		}
		count++
		batch.Queue(`
			INSERT INTO snap_disappearances (run_id, notice_id, solicitation_number, last_seen_date, disappeared_date, last_type, last_archive_type, last_archive_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, currentRunID, noticeID, solNum, lastSeen, snapshotDate, typ, archType, archDate)
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterate disappeared records: %w", err)
	}

	if batch.Len() > 0 {
		results := db.pool.SendBatch(ctx, batch)
		defer results.Close()
		for i := 0; i < batch.Len(); i++ {
			if _, err := results.Exec(); err != nil {
				logger.Warn("failed to insert snap_disappearance", "error", err)
			}
		}
	}

	return count, nil
}

func (db *DB) DetectReappearances(ctx context.Context, currentRunID uuid.UUID, snapshotDate time.Time) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE snap_disappearances d
		SET resolution = 'glitch',
		    resolution_date = $2,
		    resolution_source = 'active_csv_reappeared',
		    reappeared_date = $2
		FROM snap_csv c
		WHERE d.notice_id = c.notice_id
		  AND c.run_id = $1
		  AND d.resolution IS NULL
	`, currentRunID, snapshotDate)
	if err != nil {
		return 0, fmt.Errorf("detect reappearances: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

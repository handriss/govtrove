package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type APIUpdateProbeRecord struct {
	ID               int
	ProbedAt         time.Time
	TotalRecords     *int
	PostedFrom       *string
	SampleNoticeID   *string
	SamplePostedDate *string
	HTTPStatus       *int
	ResponseTimeMs   *int
	ErrorMessage     *string
}

func (db *DB) InsertAPIUpdateProbe(ctx context.Context, r *APIUpdateProbeRecord) (int, error) {
	query := `
		INSERT INTO api_update_probe (
			total_records, posted_from, sample_notice_id, sample_posted_date,
			http_status, response_time_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int
	err := db.pool.QueryRow(ctx, query,
		r.TotalRecords, r.PostedFrom, r.SampleNoticeID, r.SamplePostedDate,
		r.HTTPStatus, r.ResponseTimeMs, r.ErrorMessage,
	).Scan(&id)
	return id, err
}

func (db *DB) GetLatestAPIUpdateProbe(ctx context.Context) (*APIUpdateProbeRecord, error) {
	query := `
		SELECT id, probed_at, total_records, posted_from, sample_notice_id,
			sample_posted_date, http_status, response_time_ms, error_message
		FROM api_update_probe
		WHERE error_message IS NULL
		ORDER BY probed_at DESC LIMIT 1
	`
	r := &APIUpdateProbeRecord{}
	err := db.pool.QueryRow(ctx, query).Scan(
		&r.ID, &r.ProbedAt, &r.TotalRecords, &r.PostedFrom, &r.SampleNoticeID,
		&r.SamplePostedDate, &r.HTTPStatus, &r.ResponseTimeMs, &r.ErrorMessage,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return r, err
}

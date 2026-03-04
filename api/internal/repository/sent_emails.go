package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SentEmailsRepository struct {
	pool *pgxpool.Pool
}

func NewSentEmailsRepository(pool *pgxpool.Pool) *SentEmailsRepository {
	return &SentEmailsRepository{pool: pool}
}

type SentEmail struct {
	ID              string     `json:"id"`
	UserID          int        `json:"user_id"`
	ToEmail         string     `json:"to_email"`
	EmailType       string     `json:"email_type"`
	TemplateName    string     `json:"template_name"`
	TemplateData    *string    `json:"template_data"`
	Subject         string     `json:"subject"`
	ResendMessageID *string    `json:"resend_message_id"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type SentEmailWithUser struct {
	SentEmail
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

func (r *SentEmailsRepository) GetByID(ctx context.Context, id string) (*SentEmail, error) {
	var se SentEmail
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, to_email, email_type, template_name, template_data, subject, resend_message_id, status, created_at, updated_at
		 FROM sent_emails WHERE id = $1`, id,
	).Scan(&se.ID, &se.UserID, &se.ToEmail, &se.EmailType, &se.TemplateName, &se.TemplateData,
		&se.Subject, &se.ResendMessageID, &se.Status, &se.CreatedAt, &se.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting sent email: %w", err)
	}
	return &se, nil
}

func (r *SentEmailsRepository) UpdateStatusByResendID(ctx context.Context, resendMessageID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sent_emails SET status = $1, updated_at = now() WHERE resend_message_id = $2`,
		status, resendMessageID,
	)
	if err != nil {
		return fmt.Errorf("updating sent email status: %w", err)
	}
	return nil
}

func (r *SentEmailsRepository) GetUserIDByResendID(ctx context.Context, resendMessageID string) (int, error) {
	var userID int
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM sent_emails WHERE resend_message_id = $1`, resendMessageID,
	).Scan(&userID)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting user_id by resend_id: %w", err)
	}
	return userID, nil
}

func (r *SentEmailsRepository) List(ctx context.Context, page, limit int) ([]SentEmailWithUser, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sent_emails`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting sent emails: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx,
		`SELECT se.id, se.user_id, se.to_email, se.email_type, se.template_name, se.template_data,
			se.subject, se.resend_message_id, se.status, se.created_at, se.updated_at,
			u.email, COALESCE(u.first_name || ' ' || u.last_name, u.email)
		 FROM sent_emails se
		 JOIN users u ON u.id = se.user_id
		 ORDER BY se.created_at DESC
		 LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("listing sent emails: %w", err)
	}
	defer rows.Close()

	var results []SentEmailWithUser
	for rows.Next() {
		var se SentEmailWithUser
		if err := rows.Scan(&se.ID, &se.UserID, &se.ToEmail, &se.EmailType, &se.TemplateName, &se.TemplateData,
			&se.Subject, &se.ResendMessageID, &se.Status, &se.CreatedAt, &se.UpdatedAt,
			&se.UserEmail, &se.UserName); err != nil {
			return nil, 0, fmt.Errorf("scanning sent email: %w", err)
		}
		results = append(results, se)
	}
	return results, total, rows.Err()
}

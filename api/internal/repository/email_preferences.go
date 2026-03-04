package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailPreferencesRepository struct {
	pool *pgxpool.Pool
}

func NewEmailPreferencesRepository(pool *pgxpool.Pool) *EmailPreferencesRepository {
	return &EmailPreferencesRepository{pool: pool}
}

type EmailPreferences struct {
	UserID           int        `json:"user_id"`
	SearchAlerts     bool       `json:"search_alerts"`
	OpportunityAlerts bool      `json:"opportunity_alerts"`
	UnsubscribedAt   *time.Time `json:"unsubscribed_at"`
	UnsubscribeReason *string   `json:"unsubscribe_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type UserEmailPreference struct {
	UserID            int        `json:"user_id"`
	Email             string     `json:"email"`
	FirstName         string     `json:"first_name"`
	LastName          string     `json:"last_name"`
	SearchAlerts      bool       `json:"search_alerts"`
	OpportunityAlerts bool       `json:"opportunity_alerts"`
	UnsubscribedAt    *time.Time `json:"unsubscribed_at"`
	UnsubscribeReason *string    `json:"unsubscribe_reason"`
}

func (r *EmailPreferencesRepository) GetByUserID(ctx context.Context, userID int) (*EmailPreferences, error) {
	var ep EmailPreferences
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, search_alerts, opportunity_alerts, unsubscribed_at, unsubscribe_reason, created_at, updated_at
		 FROM email_preferences WHERE user_id = $1`, userID,
	).Scan(&ep.UserID, &ep.SearchAlerts, &ep.OpportunityAlerts, &ep.UnsubscribedAt, &ep.UnsubscribeReason, &ep.CreatedAt, &ep.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting email preferences: %w", err)
	}
	return &ep, nil
}

func (r *EmailPreferencesRepository) Upsert(ctx context.Context, userID int, searchAlerts, opportunityAlerts bool) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_preferences (user_id, search_alerts, opportunity_alerts)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET
			search_alerts = EXCLUDED.search_alerts,
			opportunity_alerts = EXCLUDED.opportunity_alerts,
			updated_at = now()`,
		userID, searchAlerts, opportunityAlerts,
	)
	if err != nil {
		return fmt.Errorf("upserting email preferences: %w", err)
	}
	return nil
}

func (r *EmailPreferencesRepository) CreateDefaults(ctx context.Context, userID int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("creating default email preferences: %w", err)
	}
	return nil
}

func (r *EmailPreferencesRepository) Unsubscribe(ctx context.Context, userID int, reason string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_preferences (user_id, search_alerts, opportunity_alerts, unsubscribed_at, unsubscribe_reason)
		 VALUES ($1, false, false, now(), $2)
		 ON CONFLICT (user_id) DO UPDATE SET
			search_alerts = false,
			opportunity_alerts = false,
			unsubscribed_at = now(),
			unsubscribe_reason = $2,
			updated_at = now()`,
		userID, reason,
	)
	if err != nil {
		return fmt.Errorf("unsubscribing user %d: %w", userID, err)
	}
	return nil
}

func (r *EmailPreferencesRepository) Resubscribe(ctx context.Context, userID int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_preferences SET
			search_alerts = true,
			opportunity_alerts = true,
			unsubscribed_at = NULL,
			unsubscribe_reason = NULL,
			updated_at = now()
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("resubscribing user %d: %w", userID, err)
	}
	return nil
}

func (r *EmailPreferencesRepository) ListAllWithUsers(ctx context.Context) ([]UserEmailPreference, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, u.first_name, u.last_name,
			COALESCE(ep.search_alerts, true),
			COALESCE(ep.opportunity_alerts, true),
			ep.unsubscribed_at,
			ep.unsubscribe_reason
		 FROM users u
		 LEFT JOIN email_preferences ep ON ep.user_id = u.id
		 ORDER BY u.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing email preferences: %w", err)
	}
	defer rows.Close()

	var results []UserEmailPreference
	for rows.Next() {
		var r UserEmailPreference
		if err := rows.Scan(&r.UserID, &r.Email, &r.FirstName, &r.LastName,
			&r.SearchAlerts, &r.OpportunityAlerts, &r.UnsubscribedAt, &r.UnsubscribeReason); err != nil {
			return nil, fmt.Errorf("scanning email preference: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

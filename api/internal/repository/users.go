package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

const userColumns = `id, workos_id, email, first_name, last_name, plan, free_forever, stripe_customer_id,
	subscription_id, subscription_status, cancel_at_period_end, current_period_end,
	gift_expires_at, created_at, updated_at`

func scanUser(row pgx.Row, u *models.User) error {
	return row.Scan(
		&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName,
		&u.Plan, &u.FreeForever, &u.StripeCustomerID,
		&u.SubscriptionID, &u.SubscriptionStatus, &u.CancelAtPeriodEnd, &u.CurrentPeriodEnd,
		&u.GiftExpiresAt, &u.CreatedAt, &u.UpdatedAt,
	)
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type UpsertResult struct {
	User  *models.User
	IsNew bool
}

func (r *UserRepository) Upsert(ctx context.Context, input *models.UpsertUserInput) (*UpsertResult, error) {
	// xmax = 0 means the row was inserted (not updated)
	query := `
		INSERT INTO users (workos_id, email, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workos_id) DO UPDATE SET
			email = EXCLUDED.email,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = NOW()
		RETURNING ` + userColumns + `, (xmax = 0) AS is_new
	`

	var u models.User
	var isNew bool
	err := r.pool.QueryRow(ctx, query,
		input.WorkOSID,
		input.Email,
		input.FirstName,
		input.LastName,
	).Scan(
		&u.ID, &u.WorkOSID, &u.Email, &u.FirstName, &u.LastName,
		&u.Plan, &u.FreeForever, &u.StripeCustomerID,
		&u.SubscriptionID, &u.SubscriptionStatus, &u.CancelAtPeriodEnd, &u.CurrentPeriodEnd,
		&u.GiftExpiresAt, &u.CreatedAt, &u.UpdatedAt, &isNew,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting user: %w", err)
	}
	return &UpsertResult{User: &u, IsNew: isNew}, nil
}

func (r *UserRepository) GetByWorkOSID(ctx context.Context, workosID string) (*models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE workos_id = $1`

	var u models.User
	if err := scanUser(r.pool.QueryRow(ctx, query, workosID), &u); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user by workos_id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	var u models.User
	if err := scanUser(r.pool.QueryRow(ctx, query, id), &u); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) IsAdmin(ctx context.Context, workosID string) (bool, error) {
	var isAdmin bool
	err := r.pool.QueryRow(ctx, `SELECT is_admin FROM users WHERE workos_id = $1`, workosID).Scan(&isAdmin)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking admin status: %w", err)
	}
	return isAdmin, nil
}

type AdminUserRow struct {
	ID              int
	Email           string
	FirstName       string
	LastName        string
	Plan            string
	IsAdmin         bool
	FreeForever     bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PendingExport   bool
	PendingDeletion bool
}

func (r *UserRepository) ListUsers(ctx context.Context) ([]AdminUserRow, error) {
	query := `
		SELECT u.id, u.email, u.first_name, u.last_name, u.plan, u.is_admin, u.free_forever, u.created_at, u.updated_at,
			EXISTS(SELECT 1 FROM account_requests ar WHERE ar.user_id = u.id AND ar.request_type = 'data_export' AND ar.status = 'pending') AS pending_export,
			EXISTS(SELECT 1 FROM account_requests ar WHERE ar.user_id = u.id AND ar.request_type = 'account_deletion' AND ar.status = 'pending') AS pending_deletion
		FROM users u ORDER BY u.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []AdminUserRow
	for rows.Next() {
		var u AdminUserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Plan, &u.IsAdmin, &u.FreeForever, &u.CreatedAt, &u.UpdatedAt, &u.PendingExport, &u.PendingDeletion); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

type UserExport struct {
	User               json.RawMessage `json:"user"`
	SavedOpportunities json.RawMessage `json:"saved_opportunities"`
	SavedSearches      json.RawMessage `json:"saved_searches"`
	AccountRequests    json.RawMessage `json:"account_requests"`
	ContactMessages    json.RawMessage `json:"contact_messages"`
	Notifications      json.RawMessage `json:"notifications"`
	EmailPreferences   json.RawMessage `json:"email_preferences"`
}

func (r *UserRepository) ExportUserData(ctx context.Context, userID int, userEmail string) (*UserExport, error) {
	var exp UserExport

	queryJSON := func(dest *json.RawMessage, query string, args ...any) error {
		return r.pool.QueryRow(ctx, query, args...).Scan(dest)
	}

	if err := queryJSON(&exp.User, `
		SELECT row_to_json(u) FROM (
			SELECT id, email, first_name, last_name, plan, free_forever, created_at, updated_at
			FROM users WHERE id = $1
		) u`, userID); err != nil {
		return nil, fmt.Errorf("exporting user: %w", err)
	}

	if err := queryJSON(&exp.SavedOpportunities, `
		SELECT COALESCE(json_agg(row_to_json(t)), '[]') FROM (
			SELECT so.opportunity_id, o.title, o.solicitation_number, so.notes, so.created_at
			FROM saved_opportunities so
			LEFT JOIN opportunities o ON o.id = so.opportunity_id
			WHERE so.user_id = $1
			ORDER BY so.created_at DESC
		) t`, userID); err != nil {
		return nil, fmt.Errorf("exporting saved_opportunities: %w", err)
	}

	if err := queryJSON(&exp.SavedSearches, `
		SELECT COALESCE(json_agg(row_to_json(t)), '[]') FROM (
			SELECT id, name, filters, created_at, updated_at
			FROM saved_searches WHERE user_id = $1
			ORDER BY updated_at DESC
		) t`, userID); err != nil {
		return nil, fmt.Errorf("exporting saved_searches: %w", err)
	}

	if err := queryJSON(&exp.AccountRequests, `
		SELECT COALESCE(json_agg(row_to_json(t)), '[]') FROM (
			SELECT id, request_type, status, created_at
			FROM account_requests WHERE user_id = $1
			ORDER BY created_at DESC
		) t`, userID); err != nil {
		return nil, fmt.Errorf("exporting account_requests: %w", err)
	}

	if err := queryJSON(&exp.ContactMessages, `
		SELECT COALESCE(json_agg(row_to_json(t)), '[]') FROM (
			SELECT id, name, subject, message, created_at
			FROM contact_messages WHERE email = $1
			ORDER BY created_at DESC
		) t`, userEmail); err != nil {
		return nil, fmt.Errorf("exporting contact_messages: %w", err)
	}

	if err := queryJSON(&exp.Notifications, `
		SELECT COALESCE(json_agg(row_to_json(t)), '[]') FROM (
			SELECT id, update_type, details, is_read, created_at
			FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC
		) t`, userID); err != nil {
		return nil, fmt.Errorf("exporting notifications: %w", err)
	}

	// email_preferences may not exist — use a nullable scan
	var epRaw *json.RawMessage
	if err := r.pool.QueryRow(ctx, `
		SELECT row_to_json(t) FROM (
			SELECT search_alerts, opportunity_alerts, unsubscribed_at, unsubscribe_reason
			FROM email_preferences WHERE user_id = $1
		) t`, userID).Scan(&epRaw); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("exporting email_preferences: %w", err)
	}
	if epRaw != nil {
		exp.EmailPreferences = *epRaw
	}

	return &exp, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, userID int, email string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM account_requests WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting account_requests: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE search_events SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("nullifying search_events: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE contact_messages SET name = 'deleted', email = 'deleted@deleted.invalid' WHERE email = $1`, email); err != nil {
		return fmt.Errorf("anonymizing contact_messages: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM mcp_usage WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting mcp_usage: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM invite_link_redemptions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting invite_link_redemptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM gift_code_redemptions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting gift_code_redemptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE stripe_customer_id = $1`

	var u models.User
	if err := scanUser(r.pool.QueryRow(ctx, query, customerID), &u); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user by stripe_customer_id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) SetStripeCustomerID(ctx context.Context, userID int, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`,
		customerID, userID,
	)
	if err != nil {
		return fmt.Errorf("setting stripe_customer_id: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateSubscription(ctx context.Context, userID int, plan string, subID *string, status *string, cancelAtPeriodEnd bool, periodEnd *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET plan = $1, subscription_id = $2, subscription_status = $3,
			cancel_at_period_end = $4, current_period_end = $5, updated_at = NOW()
		WHERE id = $6`,
		plan, subID, status, cancelAtPeriodEnd, periodEnd, userID,
	)
	if err != nil {
		return fmt.Errorf("updating subscription: %w", err)
	}
	return nil
}

func (r *UserRepository) SetFreeForever(ctx context.Context, userID int, freeForever bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET free_forever = $1, updated_at = NOW() WHERE id = $2`,
		freeForever, userID,
	)
	if err != nil {
		return fmt.Errorf("setting free_forever: %w", err)
	}
	return nil
}

func (r *UserRepository) SetGiftExpiry(ctx context.Context, userID int, giftExpiresAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET gift_expires_at = GREATEST(gift_expires_at, $1), updated_at = NOW() WHERE id = $2`,
		giftExpiresAt, userID,
	)
	if err != nil {
		return fmt.Errorf("setting gift_expires_at: %w", err)
	}
	return nil
}

func (r *UserRepository) CheckAndRecordWebhookEvent(ctx context.Context, eventID string, eventType string) (alreadyProcessed bool, err error) {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO stripe_webhook_events (event_id, event_type) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		eventID, eventType,
	)
	if err != nil {
		return false, fmt.Errorf("recording webhook event: %w", err)
	}
	return tag.RowsAffected() == 0, nil
}

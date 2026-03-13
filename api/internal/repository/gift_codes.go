package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GiftCodeRepository struct {
	pool *pgxpool.Pool
}

func NewGiftCodeRepository(pool *pgxpool.Pool) *GiftCodeRepository {
	return &GiftCodeRepository{pool: pool}
}

type GiftCodeRow struct {
	ID              int        `json:"id"`
	Code            string     `json:"code"`
	CampaignName    string     `json:"campaign_name"`
	DurationDays    int        `json:"duration_days"`
	MaxRedemptions  int        `json:"max_redemptions"`
	RedemptionCount int        `json:"redemption_count"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	DeactivatedAt   *time.Time `json:"deactivated_at"`
}

type GiftCodeRedemptionRow struct {
	UserID       int       `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	GrantedUntil time.Time `json:"granted_until"`
	RedeemedAt   time.Time `json:"redeemed_at"`
}

func (r *GiftCodeRepository) Create(ctx context.Context, code, campaignName string, durationDays, maxRedemptions int, expiresAt *time.Time) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO gift_codes (code, campaign_name, duration_days, max_redemptions, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		code, campaignName, durationDays, maxRedemptions, expiresAt,
	).Scan(&id)
	return id, err
}

func (r *GiftCodeRepository) List(ctx context.Context) ([]GiftCodeRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT gc.id, gc.code, gc.campaign_name, gc.duration_days, gc.max_redemptions,
		        COUNT(gcr.id)::int AS redemption_count,
		        gc.expires_at, gc.created_at, gc.deactivated_at
		 FROM gift_codes gc
		 LEFT JOIN gift_code_redemptions gcr ON gcr.gift_code_id = gc.id
		 GROUP BY gc.id
		 ORDER BY gc.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GiftCodeRow
	for rows.Next() {
		var row GiftCodeRow
		if err := rows.Scan(
			&row.ID, &row.Code, &row.CampaignName, &row.DurationDays, &row.MaxRedemptions,
			&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *GiftCodeRepository) GetByCode(ctx context.Context, code string) (*GiftCodeRow, error) {
	var row GiftCodeRow
	err := r.pool.QueryRow(ctx,
		`SELECT gc.id, gc.code, gc.campaign_name, gc.duration_days, gc.max_redemptions,
		        (SELECT COUNT(*)::int FROM gift_code_redemptions WHERE gift_code_id = gc.id) AS redemption_count,
		        gc.expires_at, gc.created_at, gc.deactivated_at
		 FROM gift_codes gc
		 WHERE gc.code = $1`,
		code,
	).Scan(&row.ID, &row.Code, &row.CampaignName, &row.DurationDays, &row.MaxRedemptions,
		&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *GiftCodeRepository) GetByID(ctx context.Context, id int) (*GiftCodeRow, error) {
	var row GiftCodeRow
	err := r.pool.QueryRow(ctx,
		`SELECT gc.id, gc.code, gc.campaign_name, gc.duration_days, gc.max_redemptions,
		        (SELECT COUNT(*)::int FROM gift_code_redemptions WHERE gift_code_id = gc.id) AS redemption_count,
		        gc.expires_at, gc.created_at, gc.deactivated_at
		 FROM gift_codes gc
		 WHERE gc.id = $1`,
		id,
	).Scan(&row.ID, &row.Code, &row.CampaignName, &row.DurationDays, &row.MaxRedemptions,
		&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *GiftCodeRepository) UpdateMaxRedemptions(ctx context.Context, id, maxRedemptions int) error {
	_, err := r.pool.Exec(ctx, `UPDATE gift_codes SET max_redemptions = $1 WHERE id = $2`, maxRedemptions, id)
	return err
}

func (r *GiftCodeRepository) Deactivate(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `UPDATE gift_codes SET deactivated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *GiftCodeRepository) HasUserRedeemed(ctx context.Context, giftCodeID, userID int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM gift_code_redemptions WHERE gift_code_id = $1 AND user_id = $2)`,
		giftCodeID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *GiftCodeRepository) AddRedemption(ctx context.Context, giftCodeID, userID int, grantedUntil time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gift_code_redemptions (gift_code_id, user_id, granted_until)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (gift_code_id, user_id) DO NOTHING`,
		giftCodeID, userID, grantedUntil,
	)
	return err
}

func (r *GiftCodeRepository) GetRedemptions(ctx context.Context, giftCodeID int) ([]GiftCodeRedemptionRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT gcr.user_id, u.email,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS name,
		        gcr.granted_until, gcr.created_at
		 FROM gift_code_redemptions gcr
		 JOIN users u ON u.id = gcr.user_id
		 WHERE gcr.gift_code_id = $1
		 ORDER BY gcr.created_at DESC`,
		giftCodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GiftCodeRedemptionRow
	for rows.Next() {
		var row GiftCodeRedemptionRow
		if err := rows.Scan(&row.UserID, &row.Email, &row.Name, &row.GrantedUntil, &row.RedeemedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *GiftCodeRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM gift_codes WHERE code = $1)`,
		code,
	).Scan(&exists)
	return exists, err
}

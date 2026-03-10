package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InviteLinkRepository struct {
	pool *pgxpool.Pool
}

func NewInviteLinkRepository(pool *pgxpool.Pool) *InviteLinkRepository {
	return &InviteLinkRepository{pool: pool}
}

type InviteLinkRow struct {
	ID              int        `json:"id"`
	Code            string     `json:"code"`
	StripePromoID   string     `json:"stripe_promo_id"`
	CampaignName    string     `json:"campaign_name"`
	MaxRedemptions  int        `json:"max_redemptions"`
	RedemptionCount int        `json:"redemption_count"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	DeactivatedAt   *time.Time `json:"deactivated_at"`
}

type InviteLinkRedemptionRow struct {
	UserID     int       `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	RedeemedAt time.Time `json:"redeemed_at"`
}

func (r *InviteLinkRepository) Create(ctx context.Context, code, stripePromoID, campaignName string, maxRedemptions int, expiresAt *time.Time) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO invite_links (code, stripe_promo_id, campaign_name, max_redemptions, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		code, stripePromoID, campaignName, maxRedemptions, expiresAt,
	).Scan(&id)
	return id, err
}

func (r *InviteLinkRepository) List(ctx context.Context) ([]InviteLinkRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT il.id, il.code, il.stripe_promo_id, il.campaign_name, il.max_redemptions,
		        COUNT(ilr.id)::int AS redemption_count,
		        il.expires_at, il.created_at, il.deactivated_at
		 FROM invite_links il
		 LEFT JOIN invite_link_redemptions ilr ON ilr.invite_link_id = il.id
		 GROUP BY il.id
		 ORDER BY il.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []InviteLinkRow
	for rows.Next() {
		var row InviteLinkRow
		if err := rows.Scan(
			&row.ID, &row.Code, &row.StripePromoID, &row.CampaignName, &row.MaxRedemptions,
			&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *InviteLinkRepository) GetByCode(ctx context.Context, code string) (*InviteLinkRow, error) {
	var row InviteLinkRow
	err := r.pool.QueryRow(ctx,
		`SELECT il.id, il.code, il.stripe_promo_id, il.campaign_name, il.max_redemptions,
		        (SELECT COUNT(*)::int FROM invite_link_redemptions WHERE invite_link_id = il.id) AS redemption_count,
		        il.expires_at, il.created_at, il.deactivated_at
		 FROM invite_links il
		 WHERE il.code = $1`,
		code,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.CampaignName, &row.MaxRedemptions,
		&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *InviteLinkRepository) GetByStripePromoID(ctx context.Context, stripePromoID string) (*InviteLinkRow, error) {
	var row InviteLinkRow
	err := r.pool.QueryRow(ctx,
		`SELECT il.id, il.code, il.stripe_promo_id, il.campaign_name, il.max_redemptions,
		        (SELECT COUNT(*)::int FROM invite_link_redemptions WHERE invite_link_id = il.id) AS redemption_count,
		        il.expires_at, il.created_at, il.deactivated_at
		 FROM invite_links il
		 WHERE il.stripe_promo_id = $1`,
		stripePromoID,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.CampaignName, &row.MaxRedemptions,
		&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *InviteLinkRepository) GetByID(ctx context.Context, id int) (*InviteLinkRow, error) {
	var row InviteLinkRow
	err := r.pool.QueryRow(ctx,
		`SELECT il.id, il.code, il.stripe_promo_id, il.campaign_name, il.max_redemptions,
		        (SELECT COUNT(*)::int FROM invite_link_redemptions WHERE invite_link_id = il.id) AS redemption_count,
		        il.expires_at, il.created_at, il.deactivated_at
		 FROM invite_links il
		 WHERE il.id = $1`,
		id,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.CampaignName, &row.MaxRedemptions,
		&row.RedemptionCount, &row.ExpiresAt, &row.CreatedAt, &row.DeactivatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *InviteLinkRepository) UpdateMaxRedemptions(ctx context.Context, id, maxRedemptions int) error {
	_, err := r.pool.Exec(ctx, `UPDATE invite_links SET max_redemptions = $1 WHERE id = $2`, maxRedemptions, id)
	return err
}

func (r *InviteLinkRepository) Deactivate(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `UPDATE invite_links SET deactivated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *InviteLinkRepository) AddRedemption(ctx context.Context, inviteLinkID, userID int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invite_link_redemptions (invite_link_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (invite_link_id, user_id) DO NOTHING`,
		inviteLinkID, userID,
	)
	return err
}

func (r *InviteLinkRepository) GetRedemptions(ctx context.Context, inviteLinkID int) ([]InviteLinkRedemptionRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ilr.user_id, u.email,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS name,
		        ilr.created_at
		 FROM invite_link_redemptions ilr
		 JOIN users u ON u.id = ilr.user_id
		 WHERE ilr.invite_link_id = $1
		 ORDER BY ilr.created_at DESC`,
		inviteLinkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []InviteLinkRedemptionRow
	for rows.Next() {
		var row InviteLinkRedemptionRow
		if err := rows.Scan(&row.UserID, &row.Email, &row.Name, &row.RedeemedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *InviteLinkRepository) CodeExistsInPromos(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM promo_codes WHERE code = $1)`,
		code,
	).Scan(&exists)
	return exists, err
}

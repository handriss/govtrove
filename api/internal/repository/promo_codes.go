package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PromoCodeRepository struct {
	pool *pgxpool.Pool
}

func NewPromoCodeRepository(pool *pgxpool.Pool) *PromoCodeRepository {
	return &PromoCodeRepository{pool: pool}
}

type PromoCodeRow struct {
	ID            int        `json:"id"`
	Code          string     `json:"code"`
	StripePromoID string     `json:"stripe_promo_id"`
	ForUserID     *int       `json:"for_user_id"`
	ForUserEmail  *string    `json:"for_user_email"`
	ForUserName   *string    `json:"for_user_name"`
	RedeemedBy    *int       `json:"redeemed_by"`
	RedeemedEmail *string    `json:"redeemed_email"`
	CreatedAt     time.Time  `json:"created_at"`
	RedeemedAt    *time.Time `json:"redeemed_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at"`
}

func (r *PromoCodeRepository) Create(ctx context.Context, code, stripePromoID string, forUserID *int, expiresAt *time.Time) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO promo_codes (code, stripe_promo_id, for_user_id, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		code, stripePromoID, forUserID, expiresAt,
	).Scan(&id)
	return id, err
}

func (r *PromoCodeRepository) List(ctx context.Context) ([]PromoCodeRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pc.id, pc.code, pc.stripe_promo_id, pc.for_user_id,
		        fu.email AS for_user_email,
		        COALESCE(fu.first_name || ' ' || fu.last_name, '') AS for_user_name,
		        pc.redeemed_by, ru.email AS redeemed_email,
		        pc.created_at, pc.redeemed_at, pc.expires_at, pc.revoked_at
		 FROM promo_codes pc
		 LEFT JOIN users fu ON fu.id = pc.for_user_id
		 LEFT JOIN users ru ON ru.id = pc.redeemed_by
		 ORDER BY pc.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PromoCodeRow
	for rows.Next() {
		var row PromoCodeRow
		if err := rows.Scan(
			&row.ID, &row.Code, &row.StripePromoID, &row.ForUserID,
			&row.ForUserEmail, &row.ForUserName,
			&row.RedeemedBy, &row.RedeemedEmail,
			&row.CreatedAt, &row.RedeemedAt, &row.ExpiresAt, &row.RevokedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *PromoCodeRepository) GetByCode(ctx context.Context, code string) (*PromoCodeRow, error) {
	var row PromoCodeRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, stripe_promo_id, for_user_id, redeemed_by, created_at, redeemed_at, expires_at
		 FROM promo_codes WHERE code = $1 AND revoked_at IS NULL`,
		code,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.ForUserID, &row.RedeemedBy, &row.CreatedAt, &row.RedeemedAt, &row.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *PromoCodeRepository) GetByStripePromoID(ctx context.Context, stripePromoID string) (*PromoCodeRow, error) {
	var row PromoCodeRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, stripe_promo_id, for_user_id, redeemed_by, created_at, redeemed_at, expires_at
		 FROM promo_codes WHERE stripe_promo_id = $1`,
		stripePromoID,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.ForUserID, &row.RedeemedBy, &row.CreatedAt, &row.RedeemedAt, &row.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *PromoCodeRepository) MarkRedeemed(ctx context.Context, promoCodeID, userID int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE promo_codes SET redeemed_by = $1, redeemed_at = NOW()
		 WHERE id = $2 AND redeemed_by IS NULL`,
		userID, promoCodeID,
	)
	return err
}

func (r *PromoCodeRepository) GetByID(ctx context.Context, id int) (*PromoCodeRow, error) {
	var row PromoCodeRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, stripe_promo_id, for_user_id, redeemed_by, created_at, redeemed_at, expires_at, revoked_at
		 FROM promo_codes WHERE id = $1`,
		id,
	).Scan(&row.ID, &row.Code, &row.StripePromoID, &row.ForUserID, &row.RedeemedBy, &row.CreatedAt, &row.RedeemedAt, &row.ExpiresAt, &row.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

func (r *PromoCodeRepository) Revoke(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `UPDATE promo_codes SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

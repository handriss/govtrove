package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type ContactRepository struct {
	pool *pgxpool.Pool
}

func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{pool: pool}
}

func (r *ContactRepository) Create(ctx context.Context, msg *models.ContactMessage) error {
	query := `
		INSERT INTO contact_messages (name, email, subject, message, ip_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		msg.Name,
		msg.Email,
		msg.Subject,
		msg.Message,
		msg.IPAddress,
	).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting contact message: %w", err)
	}

	return nil
}

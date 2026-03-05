package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/models"
)

type SavedOpportunityRepository struct {
	pool *pgxpool.Pool
}

func NewSavedOpportunityRepository(pool *pgxpool.Pool) *SavedOpportunityRepository {
	return &SavedOpportunityRepository{pool: pool}
}

func (r *SavedOpportunityRepository) ListIDs(ctx context.Context, userID int) ([]int, error) {
	query := `SELECT opportunity_id FROM saved_opportunities WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing saved opportunity ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning saved opportunity id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SavedOpportunityRepository) ListWithDetails(ctx context.Context, userID int, sort string, activeOnly bool, page, limit int) ([]models.SavedOpportunityDetail, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}

	var orderBy string
	switch sort {
	case "deadline":
		orderBy = "o.response_deadline ASC NULLS LAST, so.created_at DESC"
	default:
		orderBy = "so.created_at DESC"
	}

	conditions := "so.user_id = $1"
	args := []any{userID}
	argNum := 2

	if activeOnly {
		conditions += " AND o.active = true"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM saved_opportunities so JOIN opportunities o ON o.id = so.opportunity_id WHERE %s`, conditions)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args[:1]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting saved opportunities: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT so.id, so.opportunity_id, so.notice_id, so.solicitation_number, so.notes, so.created_at,
		       o.title, o.description, o.type, o.department, o.posted_date, o.response_deadline,
		       o.set_aside_code, o.set_aside_description, o.naics_code, o.pop_state, o.active,
		       EXISTS(SELECT 1 FROM notifications n WHERE n.source_id = so.id AND n.user_id = so.user_id AND n.is_read = false) as has_updates
		FROM saved_opportunities so
		JOIN opportunities o ON o.id = so.opportunity_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, conditions, orderBy, argNum, argNum+1)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing saved opportunities with details: %w", err)
	}
	defer rows.Close()

	var results []models.SavedOpportunityDetail
	for rows.Next() {
		var d models.SavedOpportunityDetail
		if err := rows.Scan(
			&d.ID, &d.OpportunityID, &d.NoticeID, &d.SolicitationNumber, &d.Notes, &d.CreatedAt,
			&d.Title, &d.Description, &d.Type, &d.Department, &d.PostedDate, &d.ResponseDeadline,
			&d.SetAsideCode, &d.SetAsideDesc, &d.NAICSCode, &d.PopState, &d.Active,
			&d.HasUpdates,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning saved opportunity detail: %w", err)
		}
		results = append(results, d)
	}
	return results, total, rows.Err()
}

func (r *SavedOpportunityRepository) Add(ctx context.Context, userID, opportunityID int) error {
	query := `
		INSERT INTO saved_opportunities (user_id, opportunity_id, notice_id, solicitation_number)
		SELECT $1, $2, o.notice_id, o.solicitation_number
		FROM opportunities o WHERE o.id = $3
		ON CONFLICT (user_id, opportunity_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, userID, opportunityID, opportunityID)
	if err != nil {
		return fmt.Errorf("adding saved opportunity: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) Remove(ctx context.Context, userID, opportunityID int) error {
	query := `DELETE FROM saved_opportunities WHERE user_id = $1 AND opportunity_id = $2`
	_, err := r.pool.Exec(ctx, query, userID, opportunityID)
	if err != nil {
		return fmt.Errorf("removing saved opportunity: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) RemoveByID(ctx context.Context, id, userID int) error {
	query := `DELETE FROM saved_opportunities WHERE id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("removing saved opportunity by id: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) RemoveByOpportunityID(ctx context.Context, userID, opportunityID int) error {
	query := `DELETE FROM saved_opportunities WHERE user_id = $1 AND opportunity_id = $2`
	_, err := r.pool.Exec(ctx, query, userID, opportunityID)
	if err != nil {
		return fmt.Errorf("removing saved opportunity by opportunity_id: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) BulkAdd(ctx context.Context, userID int, opportunityIDs []int) error {
	if len(opportunityIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO saved_opportunities (user_id, opportunity_id, notice_id, solicitation_number)
		SELECT $1, o.id, o.notice_id, o.solicitation_number
		FROM opportunities o WHERE o.id = ANY($2)
		ON CONFLICT (user_id, opportunity_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, userID, opportunityIDs)
	if err != nil {
		return fmt.Errorf("bulk adding saved opportunities: %w", err)
	}
	return nil
}

func (r *SavedOpportunityRepository) UpdateNotes(ctx context.Context, id, userID int, notes string) error {
	query := `UPDATE saved_opportunities SET notes = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, id, userID, notes)
	if err != nil {
		return fmt.Errorf("updating notes: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

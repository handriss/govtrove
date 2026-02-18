package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/handriss/govtrove/api/internal/models"
)

type OpportunityRepository struct {
	pool *pgxpool.Pool
}

func NewOpportunityRepository(pool *pgxpool.Pool) *OpportunityRepository {
	return &OpportunityRepository{pool: pool}
}

func (r *OpportunityRepository) Search(ctx context.Context, params models.SearchParams) (*models.SearchResult, error) {
	query, args := r.buildSearchQuery(params)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("executing search query: %w", err)
	}
	defer rows.Close()

	var opportunities []models.OpportunityListItem
	var totalCount int

	for rows.Next() {
		var opp models.OpportunityListItem
		err := rows.Scan(
			&opp.ID,
			&opp.NoticeID,
			&opp.Title,
			&opp.Description,
			&opp.SolicitationNumber,
			&opp.Type,
			&opp.Department,
			&opp.PostedDate,
			&opp.ResponseDeadline,
			&opp.SetAsideCode,
			&opp.SetAsideDesc,
			&opp.NAICSCode,
			&opp.PopState,
			&opp.Active,
			&totalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		opportunities = append(opportunities, opp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	totalPages := (totalCount + params.Limit - 1) / params.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.SearchResult{
		Opportunities: opportunities,
		Total:         totalCount,
		Page:          params.Page,
		Limit:         params.Limit,
		TotalPages:    totalPages,
	}, nil
}

func (r *OpportunityRepository) buildSearchQuery(params models.SearchParams) (string, []any) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, "active = true")

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ websearch_to_tsquery('english', $%d)", argNum))
		args = append(args, params.Query)
		argNum++
	}

	if len(params.Types) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argNum))
		args = append(args, params.Types)
		argNum++
	}

	if params.PostedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("posted_date >= $%d", argNum))
		args = append(args, *params.PostedFrom)
		argNum++
	}

	if params.PostedTo != nil {
		conditions = append(conditions, fmt.Sprintf("posted_date <= $%d", argNum))
		args = append(args, *params.PostedTo)
		argNum++
	}

	if params.DeadlineFrom != nil {
		conditions = append(conditions, fmt.Sprintf("response_deadline >= $%d", argNum))
		args = append(args, *params.DeadlineFrom)
		argNum++
	}

	if params.DeadlineTo != nil {
		conditions = append(conditions, fmt.Sprintf("response_deadline <= $%d", argNum))
		args = append(args, *params.DeadlineTo)
		argNum++
	}

	orderClause := r.buildOrderClause(params.Sort, params.Order, params.Query != "")

	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)
	limitArg := argNum
	offsetArg := argNum + 1

	query := fmt.Sprintf(`
		SELECT
			id, notice_id, title, description, solicitation_number, type,
			department, posted_date, response_deadline,
			set_aside_code, set_aside_description, naics_code,
			pop_state, active,
			COUNT(*) OVER() as total_count
		FROM opportunities
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), orderClause, limitArg, offsetArg)

	return query, args
}

func (r *OpportunityRepository) buildOrderClause(sort, order string, hasSearch bool) string {
	if order == "" {
		order = "desc"
	}
	order = strings.ToUpper(order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	switch sort {
	case "relevance":
		if hasSearch {
			return fmt.Sprintf("ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', $1)) %s, posted_date DESC", order)
		}
		return "ORDER BY posted_date DESC"
	case "deadline":
		return fmt.Sprintf("ORDER BY response_deadline %s NULLS LAST, posted_date DESC", order)
	case "posted_date":
		fallthrough
	default:
		return fmt.Sprintf("ORDER BY posted_date %s NULLS LAST", order)
	}
}

func (r *OpportunityRepository) GetByID(ctx context.Context, id int) (*models.Opportunity, error) {
	query := `
		SELECT
			id, notice_id, title, description, solicitation_number, type, base_type,
			department, posted_date, response_deadline,
			archive_date, set_aside_code, set_aside_description,
			naics_code, classification_code,
			pop_street_address, pop_city, pop_state, pop_zip, pop_country,
			award_number, award_amount, awardee_name,
			awardee_uei, award_date, active, ui_link, data_sources,
			primary_contact_title, primary_contact_fullname, primary_contact_email,
			primary_contact_phone, primary_contact_fax,
			secondary_contact_title, secondary_contact_fullname, secondary_contact_email,
			secondary_contact_phone, secondary_contact_fax,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`

	var opp models.Opportunity
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&opp.ID,
		&opp.NoticeID,
		&opp.Title,
		&opp.Description,
		&opp.SolicitationNumber,
		&opp.Type,
		&opp.BaseType,
		&opp.Department,
		&opp.PostedDate,
		&opp.ResponseDeadline,
		&opp.ArchiveDate,
		&opp.SetAsideCode,
		&opp.SetAsideDesc,
		&opp.NAICSCode,
		&opp.ClassificationCode,
		&opp.PopStreetAddress,
		&opp.PopCity,
		&opp.PopState,
		&opp.PopZip,
		&opp.PopCountry,
		&opp.AwardNumber,
		&opp.AwardAmount,
		&opp.AwardeeName,
		&opp.AwardeeUEI,
		&opp.AwardDate,
		&opp.Active,
		&opp.UILink,
		&opp.DataSource,
		&opp.PrimaryContactTitle,
		&opp.PrimaryContactFullname,
		&opp.PrimaryContactEmail,
		&opp.PrimaryContactPhone,
		&opp.PrimaryContactFax,
		&opp.SecondaryContactTitle,
		&opp.SecondaryContactFullname,
		&opp.SecondaryContactEmail,
		&opp.SecondaryContactPhone,
		&opp.SecondaryContactFax,
		&opp.CreatedAt,
		&opp.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying opportunity: %w", err)
	}

	resources, err := r.getResourceLinks(ctx, opp.ID)
	if err != nil {
		return nil, fmt.Errorf("getting resource links: %w", err)
	}
	opp.ResourceLinks = resources

	return &opp, nil
}

func (r *OpportunityRepository) getResourceLinks(ctx context.Context, opportunityID int) ([]string, error) {
	return nil, nil
}

func (r *OpportunityRepository) GetFilterOptions(ctx context.Context) (*models.FilterOptions, error) {
	return &models.FilterOptions{}, nil
}

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

	if len(params.SetAsides) > 0 {
		conditions = append(conditions, fmt.Sprintf("set_aside_code = ANY($%d)", argNum))
		args = append(args, params.SetAsides)
		argNum++
	}

	if len(params.NAICSCodes) > 0 {
		conditions = append(conditions, fmt.Sprintf("(naics_code = ANY($%d) OR naics_codes && $%d)", argNum, argNum))
		args = append(args, params.NAICSCodes)
		argNum++
	}

	if len(params.States) > 0 {
		conditions = append(conditions, fmt.Sprintf("pop_state_code = ANY($%d)", argNum))
		args = append(args, params.States)
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
			full_parent_path_name, posted_date, response_deadline,
			set_aside_code, set_aside_description, naics_code,
			pop_state_code, active,
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
			full_parent_path_name, posted_date, response_deadline,
			archive_date, set_aside_code, set_aside_description,
			naics_code, naics_codes, classification_code,
			pop_street_address, pop_city_name, pop_state_code, pop_zip, pop_country_code,
			award_number, award_amount, awardee_name,
			awardee_uei, award_date, active, ui_link, data_source,
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
		&opp.NAICSCodes,
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
	query := `SELECT url FROM opportunity_resources WHERE opportunity_id = $1`
	rows, err := r.pool.Query(ctx, query, opportunityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		links = append(links, url)
	}
	return links, rows.Err()
}

func (r *OpportunityRepository) GetFilterOptions(ctx context.Context) (*models.FilterOptions, error) {
	options := &models.FilterOptions{}

	typesQuery := `
		SELECT type, COUNT(*) as count
		FROM opportunities
		WHERE active = true AND type IS NOT NULL AND type != ''
		GROUP BY type
		ORDER BY count DESC
	`
	typeRows, err := r.pool.Query(ctx, typesQuery)
	if err != nil {
		return nil, fmt.Errorf("querying types: %w", err)
	}
	defer typeRows.Close()

	typeLabels := map[string]string{
		"o":  "Solicitation",
		"p":  "Presolicitation",
		"k":  "Combined Synopsis/Solicitation",
		"r":  "Sources Sought",
		"g":  "Sale of Surplus Property",
		"s":  "Special Notice",
		"i":  "Intent to Bundle",
		"a":  "Award Notice",
		"u":  "Justification",
		"j":  "Justification and Approval",
	}

	for typeRows.Next() {
		var opt models.FilterOption
		if err := typeRows.Scan(&opt.Code, &opt.Count); err != nil {
			return nil, fmt.Errorf("scanning type: %w", err)
		}
		if label, ok := typeLabels[opt.Code]; ok {
			opt.Label = label
		} else {
			opt.Label = opt.Code
		}
		options.Types = append(options.Types, opt)
	}

	setAsidesQuery := `
		SELECT set_aside_code, set_aside_description, COUNT(*) as count
		FROM opportunities
		WHERE active = true AND set_aside_code IS NOT NULL AND set_aside_code != ''
		GROUP BY set_aside_code, set_aside_description
		ORDER BY count DESC
	`
	setAsideRows, err := r.pool.Query(ctx, setAsidesQuery)
	if err != nil {
		return nil, fmt.Errorf("querying set-asides: %w", err)
	}
	defer setAsideRows.Close()

	for setAsideRows.Next() {
		var opt models.FilterOption
		var desc *string
		if err := setAsideRows.Scan(&opt.Code, &desc, &opt.Count); err != nil {
			return nil, fmt.Errorf("scanning set-aside: %w", err)
		}
		if desc != nil && *desc != "" {
			opt.Label = *desc
		} else {
			opt.Label = opt.Code
		}
		options.SetAsides = append(options.SetAsides, opt)
	}

	statesQuery := `
		SELECT pop_state_code, COUNT(*) as count
		FROM opportunities
		WHERE active = true AND pop_state_code IS NOT NULL AND pop_state_code != ''
		GROUP BY pop_state_code
		ORDER BY count DESC
	`
	stateRows, err := r.pool.Query(ctx, statesQuery)
	if err != nil {
		return nil, fmt.Errorf("querying states: %w", err)
	}
	defer stateRows.Close()

	for stateRows.Next() {
		var opt models.FilterOption
		if err := stateRows.Scan(&opt.Code, &opt.Count); err != nil {
			return nil, fmt.Errorf("scanning state: %w", err)
		}
		opt.Label = opt.Code
		options.States = append(options.States, opt)
	}

	return options, nil
}

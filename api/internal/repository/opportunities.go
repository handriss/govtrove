package repository

import (
	"context"
	"encoding/json"
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

// buildFilterConditions builds WHERE conditions from search params.
// exclude skips one dimension so facet counts aren't self-filtered:
// "set_aside", "type", "department", "naics", "state"
func buildFilterConditions(params models.SearchParams, exclude string, argStart int) ([]string, []any, int) {
	var conditions []string
	var args []any
	argNum := argStart

	conditions = append(conditions, "active = true")
	conditions = append(conditions, "is_latest = true")

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ websearch_to_tsquery('english', $%d)", argNum))
		args = append(args, params.Query)
		argNum++
	}

	if len(params.Types) > 0 && exclude != "type" {
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

	if len(params.SetAsides) > 0 && exclude != "set_aside" {
		conditions = append(conditions, fmt.Sprintf("set_aside_code = ANY($%d)", argNum))
		args = append(args, params.SetAsides)
		argNum++
	}

	if len(params.NAICSCodes) > 0 && exclude != "naics" {
		conditions = append(conditions, fmt.Sprintf("naics_code = ANY($%d)", argNum))
		args = append(args, params.NAICSCodes)
		argNum++
	}

	if params.NAICSPrefix != "" && exclude != "naics" {
		conditions = append(conditions, fmt.Sprintf("naics_code LIKE $%d", argNum))
		args = append(args, params.NAICSPrefix+"%")
		argNum++
	}

	if len(params.PSCCodes) > 0 && exclude != "psc" {
		conditions = append(conditions, fmt.Sprintf("classification_code = ANY($%d)", argNum))
		args = append(args, params.PSCCodes)
		argNum++
	}

	if params.PSCPrefix != "" && exclude != "psc" {
		conditions = append(conditions, fmt.Sprintf("classification_code LIKE $%d", argNum))
		args = append(args, params.PSCPrefix+"%")
		argNum++
	}

	if params.Department != "" && exclude != "department" {
		conditions = append(conditions, fmt.Sprintf("department ILIKE $%d", argNum))
		args = append(args, "%"+params.Department+"%")
		argNum++
	}

	if len(params.States) > 0 && exclude != "state" {
		conditions = append(conditions, fmt.Sprintf("pop_state = ANY($%d)", argNum))
		args = append(args, params.States)
		argNum++
	}

	return conditions, args, argNum
}

func (r *OpportunityRepository) buildSearchQuery(params models.SearchParams) (string, []any) {
	conditions, args, argNum := buildFilterConditions(params, "", 1)

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
	case "title":
		return fmt.Sprintf("ORDER BY title %s, posted_date DESC", order)
	case "department":
		return fmt.Sprintf("ORDER BY department %s NULLS LAST, posted_date DESC", order)
	case "set_aside_code":
		return fmt.Sprintf("ORDER BY set_aside_code %s NULLS LAST, posted_date DESC", order)
	case "naics_code":
		return fmt.Sprintf("ORDER BY naics_code %s NULLS LAST, posted_date DESC", order)
	case "pop_state":
		return fmt.Sprintf("ORDER BY pop_state %s NULLS LAST, posted_date DESC", order)
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
			pop_city_code, pop_state_code, pop_country_code,
			award_number, award_amount, awardee_name,
			awardee_uei, award_date, active, ui_link, resource_links, data_sources,
			primary_contact_title, primary_contact_fullname, primary_contact_email,
			primary_contact_phone, primary_contact_fax,
			secondary_contact_title, secondary_contact_fullname, secondary_contact_email,
			secondary_contact_phone, secondary_contact_fax,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`

	var opp models.Opportunity
	var resourceLinksJSON *string
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
		&opp.PopCityCode,
		&opp.PopStateCode,
		&opp.PopCountryCode,
		&opp.AwardNumber,
		&opp.AwardAmount,
		&opp.AwardeeName,
		&opp.AwardeeUEI,
		&opp.AwardDate,
		&opp.Active,
		&opp.UILink,
		&resourceLinksJSON,
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

	if resourceLinksJSON != nil {
		json.Unmarshal([]byte(*resourceLinksJSON), &opp.ResourceLinks)
	}

	return &opp, nil
}

func (r *OpportunityRepository) GetFilterOptions(ctx context.Context) (*models.FilterOptions, error) {
	return &models.FilterOptions{}, nil
}

func (r *OpportunityRepository) GetFacetCounts(ctx context.Context, params models.SearchParams) (*models.FacetResult, error) {
	// Total count with all filters applied
	conditions, args, _ := buildFilterConditions(params, "", 1)
	where := strings.Join(conditions, " AND ")

	var total int
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", where), args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting total: %w", err)
	}

	type facetSpec struct {
		key       string
		exclude   string
		selectCol string
		labelCol  string
		limit     int
	}
	specs := []facetSpec{
		{"set_aside", "set_aside", "set_aside_code", "set_aside_description", 0},
		{"notice_type", "type", "type", "", 0},
		{"agency", "department", "department", "", 50},
		{"naics", "naics", "naics_code", "", 50},
		{"psc", "psc", "classification_code", "", 50},
	}

	facets := make(map[string][]models.FacetValue, len(specs))
	for _, s := range specs {
		vals, err := r.getFacet(ctx, params, s.exclude, s.selectCol, s.labelCol, s.limit)
		if err != nil {
			return nil, fmt.Errorf("facet %s: %w", s.key, err)
		}
		facets[s.key] = vals
	}

	return &models.FacetResult{Total: total, Facets: facets}, nil
}

func (r *OpportunityRepository) GetSolicitationHistory(ctx context.Context, opportunityID int) (*models.SolicitationHistory, error) {
	var solNum *string
	err := r.pool.QueryRow(ctx,
		"SELECT solicitation_number FROM opportunities WHERE id = $1", opportunityID,
	).Scan(&solNum)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("looking up solicitation_number: %w", err)
	}
	if solNum == nil || *solNum == "" {
		return nil, nil
	}

	var totalNotices int
	err = r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM opportunities WHERE solicitation_number = $1", *solNum,
	).Scan(&totalNotices)
	if err != nil {
		return nil, fmt.Errorf("counting notices: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, notice_id, title, type, base_type, posted_date, response_deadline,
		       award_date, award_amount, awardee_name, active, version
		FROM opportunities
		WHERE solicitation_number = $1
		ORDER BY posted_date ASC NULLS LAST, version ASC
		LIMIT 50
	`, *solNum)
	if err != nil {
		return nil, fmt.Errorf("querying sibling notices: %w", err)
	}
	defer rows.Close()

	var items []models.SolicitationHistoryItem
	for rows.Next() {
		var item models.SolicitationHistoryItem
		err := rows.Scan(
			&item.ID, &item.NoticeID, &item.Title, &item.Type, &item.BaseType,
			&item.PostedDate, &item.ResponseDeadline,
			&item.AwardDate, &item.AwardAmount, &item.AwardeeName, &item.Active,
			&item.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning notice row: %w", err)
		}
		if item.ID == opportunityID {
			item.IsCurrent = true
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating notice rows: %w", err)
	}

	return &models.SolicitationHistory{
		SolicitationNumber: *solNum,
		TotalNotices:       totalNotices,
		Notices:            items,
		Truncated:          totalNotices > 50,
	}, nil
}

func (r *OpportunityRepository) getFacet(ctx context.Context, params models.SearchParams, exclude, selectCol, labelCol string, limit int) ([]models.FacetValue, error) {
	conditions, args, _ := buildFilterConditions(params, exclude, 1)
	where := strings.Join(conditions, " AND ")

	selectExpr := selectCol
	groupBy := selectCol
	if labelCol != "" {
		selectExpr = fmt.Sprintf("%s, %s", selectCol, labelCol)
		groupBy = fmt.Sprintf("%s, %s", selectCol, labelCol)
	}

	q := fmt.Sprintf(
		"SELECT %s, COUNT(*) AS cnt FROM opportunities WHERE %s AND %s IS NOT NULL GROUP BY %s ORDER BY cnt DESC",
		selectExpr, where, selectCol, groupBy,
	)
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.FacetValue
	for rows.Next() {
		var fv models.FacetValue
		if labelCol != "" {
			err = rows.Scan(&fv.Value, &fv.Label, &fv.Count)
		} else {
			err = rows.Scan(&fv.Value, &fv.Count)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, fv)
	}
	return result, rows.Err()
}

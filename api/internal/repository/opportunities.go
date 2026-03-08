package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/handriss/govtrove/api/internal/models"
)

var quotedPhraseRe = regexp.MustCompile(`"([^"]+)"`)

func parseQuotedPhrases(query string) (phrases []string, remainder string) {
	matches := quotedPhraseRe.FindAllStringSubmatch(query, -1)
	for _, m := range matches {
		phrases = append(phrases, m[1])
	}
	remainder = strings.TrimSpace(quotedPhraseRe.ReplaceAllString(query, ""))
	return
}

// splitOR splits a query on " OR " (case-sensitive) while respecting quoted phrases.
func splitOR(query string) []string {
	var segments []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(query); i++ {
		if query[i] == '"' {
			inQuote = !inQuote
			current.WriteByte(query[i])
		} else if !inQuote && i+4 <= len(query) && query[i:i+4] == " OR " {
			if seg := strings.TrimSpace(current.String()); seg != "" {
				segments = append(segments, seg)
			}
			current.Reset()
			i += 3
		} else {
			current.WriteByte(query[i])
		}
	}

	if seg := strings.TrimSpace(current.String()); seg != "" {
		segments = append(segments, seg)
	}
	return segments
}

type OpportunityRepository struct {
	pool *pgxpool.Pool
}

func NewOpportunityRepository(pool *pgxpool.Pool) *OpportunityRepository {
	return &OpportunityRepository{pool: pool}
}

func (r *OpportunityRepository) Search(ctx context.Context, params models.SearchParams) (*models.SearchResult, error) {
	countQuery, countArgs, dataQuery, dataArgs := r.buildSearchQuery(params)

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("counting search results: %w", err)
	}

	var opportunities []models.OpportunityListItem
	if totalCount > 0 {
		rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
		if err != nil {
			return nil, fmt.Errorf("executing search query: %w", err)
		}
		defer rows.Close()

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
			)
			if err != nil {
				return nil, fmt.Errorf("scanning row: %w", err)
			}
			opportunities = append(opportunities, opp)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterating rows: %w", err)
		}
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
func buildFilterConditions(params models.SearchParams, exclude string, argStart int) ([]string, []any, int, string) {
	var conditions []string
	var args []any
	argNum := argStart
	var ftsExpr string

	conditions = append(conditions, "active = true")
	conditions = append(conditions, "is_latest = true")

	if params.Query != "" {
		segments := splitOR(params.Query)
		if len(segments) <= 1 {
			phrases, ftsQuery := parseQuotedPhrases(params.Query)
			for _, phrase := range phrases {
				conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR solicitation_number ILIKE $%d)", argNum, argNum, argNum))
				args = append(args, "%"+phrase+"%")
				argNum++
			}
			if ftsQuery != "" {
				conditions = append(conditions, fmt.Sprintf("search_vector @@ websearch_to_tsquery('english', $%d)", argNum))
				args = append(args, ftsQuery)
				ftsExpr = fmt.Sprintf("websearch_to_tsquery('english', $%d)", argNum)
				argNum++
			}
		} else {
			var orParts []string
			var ftsExprs []string

			for _, seg := range segments {
				phrases, ftsQuery := parseQuotedPhrases(seg)
				for _, phrase := range phrases {
					orParts = append(orParts, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR solicitation_number ILIKE $%d)", argNum, argNum, argNum))
					args = append(args, "%"+phrase+"%")
					argNum++
				}
				if ftsQuery != "" {
					ftsExprs = append(ftsExprs, fmt.Sprintf("websearch_to_tsquery('english', $%d)", argNum))
					args = append(args, ftsQuery)
					argNum++
				}
			}

			if len(ftsExprs) > 0 {
				combined := strings.Join(ftsExprs, " || ")
				orParts = append(orParts, fmt.Sprintf("search_vector @@ (%s)", combined))
				ftsExpr = combined
			}

			if len(orParts) > 0 {
				conditions = append(conditions, "("+strings.Join(orParts, " OR ")+")")
			}
		}
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

	if params.Department != "" && exclude != "department" && exclude != "agency" {
		conditions = append(conditions, fmt.Sprintf("department ILIKE $%d", argNum))
		args = append(args, "%"+params.Department+"%")
		argNum++
	}

	if len(params.AgencyPaths) > 0 && exclude != "agency" {
		patterns := make([]string, len(params.AgencyPaths))
		for i, p := range params.AgencyPaths {
			patterns[i] = p + "%"
		}
		conditions = append(conditions, fmt.Sprintf("full_parent_path_name LIKE ANY($%d::text[])", argNum))
		args = append(args, patterns)
		argNum++
	}

	if len(params.States) > 0 && exclude != "state" {
		conditions = append(conditions, fmt.Sprintf("pop_state = ANY($%d)", argNum))
		args = append(args, params.States)
		argNum++
	}

	if params.SolicitationNumber != "" {
		conditions = append(conditions, fmt.Sprintf("solicitation_number ILIKE $%d", argNum))
		args = append(args, "%"+params.SolicitationNumber+"%")
		argNum++
	}

	if params.PopCity != "" {
		conditions = append(conditions, fmt.Sprintf("(pop_city ILIKE $%d OR pop_zip LIKE $%d)", argNum, argNum))
		args = append(args, params.PopCity+"%")
		argNum++
	}

	return conditions, args, argNum, ftsExpr
}

func (r *OpportunityRepository) buildSearchQuery(params models.SearchParams) (countQuery string, countArgs []any, dataQuery string, dataArgs []any) {
	conditions, args, argNum, ftsExpr := buildFilterConditions(params, "", 1)
	where := strings.Join(conditions, " AND ")

	countQuery = fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", where)
	countArgs = append([]any{}, args...)

	orderClause := r.buildOrderClause(params, ftsExpr)
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)

	dataQuery = fmt.Sprintf(`
		SELECT
			id, notice_id, title, description, solicitation_number, type,
			department, posted_date, response_deadline,
			set_aside_code, set_aside_description, naics_code,
			pop_state, active
		FROM opportunities
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, where, orderClause, argNum, argNum+1)
	dataArgs = args

	return
}

func (r *OpportunityRepository) buildOrderClause(params models.SearchParams, ftsExpr string) string {
	order := params.Order
	if order == "" {
		order = "desc"
	}
	order = strings.ToUpper(order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	switch params.Sort {
	case "relevance":
		if params.Query != "" && ftsExpr != "" {
			return fmt.Sprintf("ORDER BY ts_rank(search_vector, %s) %s, posted_date DESC", ftsExpr, order)
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

func (r *OpportunityRepository) SuggestQuery(ctx context.Context, query string) (string, error) {
	// For each word in the query, find the best-matching word from the
	// top-matching title. Only replaces a word if the per-word similarity
	// exceeds 0.3 — otherwise keeps the original query word unchanged.
	var suggestion string
	err := r.pool.QueryRow(ctx, `
		WITH best AS (
			SELECT title
			FROM opportunities
			WHERE active = true AND is_latest = true AND word_similarity($1, title) > 0.4
			ORDER BY word_similarity($1, title) DESC
			LIMIT 1
		),
		query_words AS (
			SELECT ordinality, word AS qw
			FROM regexp_split_to_table($1, '\s+') WITH ORDINALITY AS t(word, ordinality)
			WHERE length(word) > 0
		),
		title_words AS (
			SELECT DISTINCT regexp_replace(word, '[^a-zA-Z0-9''-]', '', 'g') AS tw
			FROM best, regexp_split_to_table(best.title, '\s+') AS word
			WHERE length(regexp_replace(word, '[^a-zA-Z0-9''-]', '', 'g')) > 1
		),
		matched AS (
			SELECT qw.ordinality,
				CASE WHEN (SELECT similarity(qw.qw, tw) FROM title_words ORDER BY similarity(qw.qw, tw) DESC LIMIT 1) > 0.3
					THEN (SELECT lower(tw) FROM title_words ORDER BY similarity(qw.qw, tw) DESC LIMIT 1)
					ELSE qw.qw
				END AS replacement
			FROM query_words qw
		)
		SELECT string_agg(replacement, ' ' ORDER BY ordinality)
		FROM matched
	`, query).Scan(&suggestion)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(suggestion), strings.TrimSpace(query)) {
		return "", nil
	}
	return suggestion, nil
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
			created_at, updated_at,
			(SELECT id FROM pipeline.snap_csv WHERE notice_id = o.notice_id ORDER BY snapshot_date DESC LIMIT 1),
			(SELECT id FROM pipeline.snap_api WHERE notice_id = o.notice_id ORDER BY snapshot_date DESC LIMIT 1)
		FROM opportunities o
		WHERE o.id = $1
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
		&opp.SnapCSVID,
		&opp.SnapAPIID,
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
	conditions, args, _, _ := buildFilterConditions(params, "", 1)
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
		{"agency", "agency", "department", "", 50},
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
		       archive_date, award_date, award_amount, awardee_name, set_aside_code,
		       active, version, primary_contact_fullname,
		       COALESCE(jsonb_array_length(resource_links), 0),
		       LEFT(description, 500)
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
			&item.ArchiveDate, &item.AwardDate, &item.AwardAmount, &item.AwardeeName,
			&item.SetAsideCode, &item.Active, &item.Version,
			&item.ContactName, &item.ResourceCount, &item.Description,
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

	computeHistoryChanges(items)

	return &models.SolicitationHistory{
		SolicitationNumber: *solNum,
		TotalNotices:       totalNotices,
		Notices:            items,
		Truncated:          totalNotices > 50,
	}, nil
}

func computeHistoryChanges(items []models.SolicitationHistoryItem) {
	for i := 1; i < len(items); i++ {
		prev := &items[i-1]
		curr := &items[i]
		var changes []models.FieldChange

		if curr.NoticeID != prev.NoticeID && curr.Version == 1 {
			changes = append(changes, models.FieldChange{FieldName: "New notice"})
		}

		if curr.Title != prev.Title {
			old := prev.Title
			changes = append(changes, models.FieldChange{FieldName: "Title", OldValue: &old, NewValue: &curr.Title})
		}

		if ptrStr(curr.Type) != ptrStr(prev.Type) {
			changes = append(changes, models.FieldChange{FieldName: "Type", OldValue: prev.Type, NewValue: curr.Type})
		}

		if ptrStr(curr.SetAsideCode) != ptrStr(prev.SetAsideCode) {
			changes = append(changes, models.FieldChange{FieldName: "Set-aside", OldValue: prev.SetAsideCode, NewValue: curr.SetAsideCode})
		}

		if fmtTime(curr.ResponseDeadline) != fmtTime(prev.ResponseDeadline) {
			old, new := fmtTime(prev.ResponseDeadline), fmtTime(curr.ResponseDeadline)
			changes = append(changes, models.FieldChange{FieldName: "Deadline", OldValue: nilIfEmpty(old), NewValue: nilIfEmpty(new)})
		}

		if fmtTime(curr.ArchiveDate) != fmtTime(prev.ArchiveDate) {
			old, new := fmtTime(prev.ArchiveDate), fmtTime(curr.ArchiveDate)
			changes = append(changes, models.FieldChange{FieldName: "Archive date", OldValue: nilIfEmpty(old), NewValue: nilIfEmpty(new)})
		}

		if curr.Active != prev.Active {
			if curr.Active {
				v := "Reactivated"
				changes = append(changes, models.FieldChange{FieldName: v})
			} else {
				v := "Deactivated"
				changes = append(changes, models.FieldChange{FieldName: v})
			}
		}

		if ptrStr(curr.AwardeeName) != ptrStr(prev.AwardeeName) && curr.AwardeeName != nil {
			changes = append(changes, models.FieldChange{FieldName: "Awardee", OldValue: prev.AwardeeName, NewValue: curr.AwardeeName})
		}

		if fmtFloat(curr.AwardAmount) != fmtFloat(prev.AwardAmount) && curr.AwardAmount != nil {
			old, new := fmtFloat(prev.AwardAmount), fmtFloat(curr.AwardAmount)
			changes = append(changes, models.FieldChange{FieldName: "Award", OldValue: nilIfEmpty(old), NewValue: nilIfEmpty(new)})
		}

		if ptrStr(curr.ContactName) != ptrStr(prev.ContactName) {
			changes = append(changes, models.FieldChange{FieldName: "Contact", OldValue: prev.ContactName, NewValue: curr.ContactName})
		}

		if curr.ResourceCount != prev.ResourceCount {
			old := fmt.Sprintf("%d", prev.ResourceCount)
			new := fmt.Sprintf("%d", curr.ResourceCount)
			changes = append(changes, models.FieldChange{FieldName: "Attachments", OldValue: &old, NewValue: &new})
		}

		if ptrStr(curr.Description) != ptrStr(prev.Description) {
			changes = append(changes, models.FieldChange{FieldName: "Description"})
		}

		curr.Changes = changes
		items[i] = *curr
	}
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func fmtFloat(f *float64) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *f)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *OpportunityRepository) getFacet(ctx context.Context, params models.SearchParams, exclude, selectCol, labelCol string, limit int) ([]models.FacetValue, error) {
	conditions, args, _, _ := buildFilterConditions(params, exclude, 1)
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

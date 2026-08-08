package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type oppSnapshot struct {
	ID                int     `json:"id"`
	Title             *string `json:"title"`
	SolicitationNum   *string `json:"solicitation_number"`
	Department        *string `json:"department"`
	NAICSCode         *string `json:"naics_code"`
	SetAsideCode      *string `json:"set_aside_code"`
	PostedDate        *string `json:"posted_date"`
	ResponseDeadline  *string `json:"response_deadline"`
	Active            bool    `json:"active"`
}

func (h *Handler) writeSearchNotifications(ctx context.Context) (int, error) {
	if !h.acquireLock(ctx, "search_notifications") {
		h.Logger.Info("search_notifications lock held, skipping")
		return 0, nil
	}
	defer h.releaseLock(ctx, "search_notifications")

	today := time.Now().Format("2006-01-02")
	totalNotifs := 0
	offset := 0
	batchSize := 100

	for {
		rows, err := h.Pool.Query(ctx, `
			SELECT id, user_id, name, filters, last_checked_at
			FROM saved_searches
			WHERE alert_enabled = true
			ORDER BY id
			LIMIT $1 OFFSET $2
		`, batchSize, offset)
		if err != nil {
			return totalNotifs, fmt.Errorf("query saved searches: %w", err)
		}

		type searchRow struct {
			ID            int
			UserID        int
			Name          string
			Filters       json.RawMessage
			LastCheckedAt *time.Time
		}

		var searches []searchRow
		for rows.Next() {
			var s searchRow
			if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Filters, &s.LastCheckedAt); err != nil {
				rows.Close()
				return totalNotifs, fmt.Errorf("scan saved search: %w", err)
			}
			searches = append(searches, s)
		}
		rows.Close()

		if len(searches) == 0 {
			break
		}

		for _, s := range searches {
			var f savedFilters
			if err := json.Unmarshal(s.Filters, &f); err != nil {
				h.Logger.Error("unmarshal search filters", "search_id", s.ID, "error", err)
				continue
			}

			// is_current collapses amendment reposts, so a saved search notifies
			// once per solicitation instead of once per amendment.
			conditions := []string{"active = true", "is_latest = true", "is_current = true"}
			args := []any{}
			argNum := 1

			if s.LastCheckedAt != nil {
				conditions = append(conditions, fmt.Sprintf("created_at > $%d", argNum))
				args = append(args, *s.LastCheckedAt)
				argNum++
			}

			conditions, args, argNum = appendFilterConditions(conditions, args, argNum, f)
			where := strings.Join(conditions, " AND ")

			var matchCount int
			if err := h.Pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM opportunities WHERE %s", where), args...).Scan(&matchCount); err != nil {
				h.Logger.Error("count search matches", "search_id", s.ID, "error", err)
				continue
			}

			if matchCount == 0 {
				h.Pool.Exec(ctx, `UPDATE saved_searches SET last_checked_at = NOW() WHERE id = $1`, s.ID)
				continue
			}

			// Fetch up to 10 opportunity snapshots
			snapQuery := fmt.Sprintf(`
				SELECT id, title, solicitation_number, department, naics_code, set_aside_code,
					to_char(posted_date, 'YYYY-MM-DD'), to_char(response_deadline, 'YYYY-MM-DD'), active
				FROM opportunities
				WHERE %s
				ORDER BY posted_date DESC
				LIMIT 10
			`, where)

			snapRows, err := h.Pool.Query(ctx, snapQuery, args...)
			if err != nil {
				h.Logger.Error("query search snapshots", "search_id", s.ID, "error", err)
				continue
			}

			var matches []oppSnapshot
			for snapRows.Next() {
				var snap oppSnapshot
				if err := snapRows.Scan(&snap.ID, &snap.Title, &snap.SolicitationNum, &snap.Department,
					&snap.NAICSCode, &snap.SetAsideCode, &snap.PostedDate, &snap.ResponseDeadline, &snap.Active); err != nil {
					h.Logger.Error("scan snapshot", "search_id", s.ID, "error", err)
					break
				}
				matches = append(matches, snap)
			}
			snapRows.Close()

			details, _ := json.Marshal(map[string]any{
				"search_name":    s.Name,
				"search_id":     s.ID,
				"search_filters": s.Filters,
				"match_count":   matchCount,
				"matches":       matches,
			})

			groupKey := fmt.Sprintf("search:%d:%s", s.ID, today)
			sourceID := s.ID

			_, err = h.Pool.Exec(ctx, `
				INSERT INTO notifications (id, user_id, update_type, source_id, group_key, details)
				VALUES ($1, $2, 'search_matches', $3, $4, $5)
				ON CONFLICT (group_key) WHERE group_key IS NOT NULL DO UPDATE SET
					details = EXCLUDED.details,
					is_read = false,
					expires_at = NOW() + INTERVAL '90 days',
					created_at = NOW()
			`, uuid.New(), s.UserID, sourceID, groupKey, string(details))
			if err != nil {
				h.Logger.Error("upsert search notification", "search_id", s.ID, "error", err)
				continue
			}

			totalNotifs++
			h.Pool.Exec(ctx, `UPDATE saved_searches SET last_checked_at = NOW() WHERE id = $1`, s.ID)
		}

		offset += batchSize
	}

	return totalNotifs, nil
}

func (h *Handler) writeOpportunityNotifications(ctx context.Context) (int, error) {
	if !h.acquireLock(ctx, "opportunity_notifications") {
		h.Logger.Info("opportunity_notifications lock held, skipping")
		return 0, nil
	}
	defer h.releaseLock(ctx, "opportunity_notifications")

	totalNotifs := 0
	today := time.Now().Format("2006-01-02")

	// Amendments — new notice under same solicitation
	aRows, err := h.Pool.Query(ctx, `
		SELECT so.id, so.user_id, so.solicitation_number, o.id, o.title, o.solicitation_number, o.department
		FROM saved_opportunities so
		JOIN opportunities o ON o.solicitation_number = so.solicitation_number
		WHERE o.notice_id != so.notice_id
		  AND o.created_at > so.last_notified_at
		  AND so.solicitation_number IS NOT NULL
		  AND o.is_latest = true
	`)
	if err != nil {
		return 0, fmt.Errorf("query amendments: %w", err)
	}
	defer aRows.Close()

	type amendRow struct {
		SavedOppID int
		UserID     int
		SolNumber  string
		OppID      int
		Title      *string
		SolNum     *string
		Department *string
	}

	var amendments []amendRow
	for aRows.Next() {
		var a amendRow
		if err := aRows.Scan(&a.SavedOppID, &a.UserID, &a.SolNumber, &a.OppID, &a.Title, &a.SolNum, &a.Department); err != nil {
			return 0, fmt.Errorf("scan amendment: %w", err)
		}
		amendments = append(amendments, a)
	}

	for _, a := range amendments {
		details, _ := json.Marshal(map[string]any{
			"opportunity_id":      a.OppID,
			"title":               a.Title,
			"solicitation_number": a.SolNum,
			"department":          a.Department,
			"change_type":         "amendment",
			"changes":             []map[string]any{{"field": "notice_id", "new": "New amendment posted"}},
		})

		groupKey := fmt.Sprintf("opp:%d:%s", a.SavedOppID, today)

		_, err := h.Pool.Exec(ctx, `
			INSERT INTO notifications (id, user_id, update_type, source_id, group_key, details)
			VALUES ($1, $2, 'opportunity_update', $3, $4, $5)
			ON CONFLICT (group_key) WHERE group_key IS NOT NULL DO UPDATE SET
				details = EXCLUDED.details,
				is_read = false,
				expires_at = NOW() + INTERVAL '90 days',
				created_at = NOW()
		`, uuid.New(), a.UserID, a.SavedOppID, groupKey, string(details))
		if err != nil {
			h.Logger.Error("upsert amendment notification", "error", err)
			continue
		}
		totalNotifs++
		h.Pool.Exec(ctx, `UPDATE saved_opportunities SET last_notified_at = NOW() WHERE id = $1`, a.SavedOppID)
	}

	// In-place changes — diff curr vs prev version
	cRows, err := h.Pool.Query(ctx, `
		SELECT so.id, so.user_id, so.notice_id, o.id, o.title, o.solicitation_number, o.department,
			curr.title, curr.response_deadline, curr.archive_date, curr.description, curr.set_aside_code, curr.award_amount, curr.resource_links,
			prev.title, prev.response_deadline, prev.archive_date, prev.description, prev.set_aside_code, prev.award_amount, prev.resource_links
		FROM saved_opportunities so
		JOIN opportunities o ON o.notice_id = so.notice_id AND o.is_latest = true
		JOIN opportunities curr ON curr.notice_id = so.notice_id AND curr.is_latest = true AND curr.version > 1
		JOIN opportunities prev ON prev.notice_id = so.notice_id AND prev.version = curr.version - 1
		WHERE curr.created_at > so.last_notified_at
	`)
	if err != nil {
		return totalNotifs, fmt.Errorf("query version changes: %w", err)
	}
	defer cRows.Close()

	for cRows.Next() {
		var (
			savedOppID int
			userID     int
			noticeID   string
			oppID      int
			oppTitle   *string
			oppSolNum  *string
			oppDept    *string
			r          versionChangeRow
		)
		if err := cRows.Scan(
			&savedOppID, &userID, &noticeID, &oppID, &oppTitle, &oppSolNum, &oppDept,
			&r.NewTitle, &r.NewResponseDeadline, &r.NewArchiveDate, &r.NewDescription, &r.NewSetAsideCode, &r.NewAwardAmount, &r.NewResourceLinks,
			&r.OldTitle, &r.OldResponseDeadline, &r.OldArchiveDate, &r.OldDescription, &r.OldSetAsideCode, &r.OldAwardAmount, &r.OldResourceLinks,
		); err != nil {
			return totalNotifs, fmt.Errorf("scan version change: %w", err)
		}

		var diffs []map[string]any

		if !strPtrEqual(r.OldTitle, r.NewTitle) {
			diffs = append(diffs, diffPtrs("title", r.OldTitle, r.NewTitle))
		}
		if !timePtrEqual(r.OldResponseDeadline, r.NewResponseDeadline) {
			diffs = append(diffs, diffTimePtrs("response_deadline", r.OldResponseDeadline, r.NewResponseDeadline))
		}
		if !timePtrEqual(r.OldArchiveDate, r.NewArchiveDate) {
			diffs = append(diffs, diffTimePtrs("archive_date", r.OldArchiveDate, r.NewArchiveDate))
		}
		if !strPtrEqual(r.OldDescription, r.NewDescription) {
			diffs = append(diffs, map[string]any{"field": "description"})
		}
		if !strPtrEqual(r.OldSetAsideCode, r.NewSetAsideCode) {
			diffs = append(diffs, diffPtrs("set_aside_code", r.OldSetAsideCode, r.NewSetAsideCode))
		}
		if !floatPtrEqual(r.OldAwardAmount, r.NewAwardAmount) {
			diffs = append(diffs, diffFloatPtrs("award_amount", r.OldAwardAmount, r.NewAwardAmount))
		}
		if added, removed := diffResourceLinks(r.OldResourceLinks, r.NewResourceLinks); len(added) > 0 || len(removed) > 0 {
			diffs = append(diffs, map[string]any{
				"field":   "resource_links",
				"added":   added,
				"removed": removed,
			})
		}

		if len(diffs) == 0 {
			continue
		}

		details, _ := json.Marshal(map[string]any{
			"opportunity_id":      oppID,
			"title":               oppTitle,
			"solicitation_number": oppSolNum,
			"department":          oppDept,
			"change_type":         "field_change",
			"changes":             diffs,
		})

		groupKey := fmt.Sprintf("opp:%d:%s", savedOppID, today)

		_, err := h.Pool.Exec(ctx, `
			INSERT INTO notifications (id, user_id, update_type, source_id, group_key, details)
			VALUES ($1, $2, 'opportunity_update', $3, $4, $5)
			ON CONFLICT (group_key) WHERE group_key IS NOT NULL DO UPDATE SET
				details = EXCLUDED.details,
				is_read = false,
				expires_at = NOW() + INTERVAL '90 days',
				created_at = NOW()
		`, uuid.New(), userID, savedOppID, groupKey, string(details))
		if err != nil {
			h.Logger.Error("upsert change notification", "error", err)
			continue
		}
		totalNotifs++
		h.Pool.Exec(ctx, `UPDATE saved_opportunities SET last_notified_at = NOW() WHERE id = $1`, savedOppID)
	}

	return totalNotifs, nil
}

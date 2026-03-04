package main

import (
	"context"
	"fmt"
	"time"
)

type notificationRow struct {
	ID         string
	UserID     int
	UpdateType string
	Details    string
	CreatedAt  time.Time
}

func (h *Handler) sendDigestEmails(ctx context.Context) (int, error) {
	if h.Email == nil {
		return 0, nil
	}

	rows, err := h.Pool.Query(ctx, `
		SELECT n.id, n.user_id, n.update_type, n.details, n.created_at
		FROM notifications n
		WHERE n.created_at > NOW() - INTERVAL '20 minutes'
		AND NOT EXISTS (
			SELECT 1 FROM sent_emails se
			WHERE se.user_id = n.user_id
			AND se.email_type = 'digest'
			AND se.created_at > NOW() - INTERVAL '23 hours'
		)
		ORDER BY n.user_id, n.update_type, n.created_at
	`)
	if err != nil {
		return 0, fmt.Errorf("query recent notifications: %w", err)
	}
	defer rows.Close()

	byUser := map[int][]notificationRow{}
	for rows.Next() {
		var n notificationRow
		if err := rows.Scan(&n.ID, &n.UserID, &n.UpdateType, &n.Details, &n.CreatedAt); err != nil {
			return 0, fmt.Errorf("scan notification: %w", err)
		}
		byUser[n.UserID] = append(byUser[n.UserID], n)
	}

	sent := 0
	for userID, notifs := range byUser {
		var email, firstName string
		var searchAlerts, oppAlerts bool
		var unsubscribedAt *time.Time

		err := h.Pool.QueryRow(ctx, `
			SELECT u.email, COALESCE(u.first_name, ''),
				COALESCE(ep.search_alerts, true), COALESCE(ep.opportunity_alerts, true), ep.unsubscribed_at
			FROM users u
			LEFT JOIN email_preferences ep ON ep.user_id = u.id
			WHERE u.id = $1
		`, userID).Scan(&email, &firstName, &searchAlerts, &oppAlerts, &unsubscribedAt)
		if err != nil {
			h.Logger.Warn("skip user: can't fetch email/prefs", "user_id", userID, "error", err)
			continue
		}

		if unsubscribedAt != nil {
			continue
		}

		var searchNotifs, oppNotifs []notificationRow
		for _, n := range notifs {
			switch n.UpdateType {
			case "search_matches":
				if searchAlerts {
					searchNotifs = append(searchNotifs, n)
				}
			case "opportunity_update":
				if oppAlerts {
					oppNotifs = append(oppNotifs, n)
				}
			}
		}

		if len(searchNotifs) == 0 && len(oppNotifs) == 0 {
			continue
		}

		if err := h.Email.SendDigest(ctx, userID, email, firstName, searchNotifs, oppNotifs); err != nil {
			h.Logger.Error("send digest failed", "user_id", userID, "error", err)
			continue
		}

		sent++
		h.Logger.Info("digest sent", "user_id", userID, "search", len(searchNotifs), "opp", len(oppNotifs))
	}

	return sent, nil
}

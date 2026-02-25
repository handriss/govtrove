package database

import (
	"context"
	"fmt"
)

func (db *DB) DeleteOldSearchEvents(ctx context.Context, days int) (int64, error) {
	tag, err := db.pool.Exec(ctx,
		`DELETE FROM search_events WHERE created_at < NOW() - INTERVAL '1 day' * $1`,
		days,
	)
	if err != nil {
		return 0, fmt.Errorf("deleting old search events: %w", err)
	}
	return tag.RowsAffected(), nil
}

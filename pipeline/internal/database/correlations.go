package database

import (
	"context"
	"fmt"
)

func (db *DB) RefreshCodeCorrelations(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY code_correlations`)
	if err != nil {
		return fmt.Errorf("refresh code correlations: %w", err)
	}
	return nil
}

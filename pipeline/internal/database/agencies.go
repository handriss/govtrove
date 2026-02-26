package database

import (
	"context"
	"fmt"
)

func (db *DB) RefreshAgencies(ctx context.Context) (int, error) {
	tag, err := db.pool.Exec(ctx, `
		WITH path_counts AS (
			SELECT full_parent_path_name AS path, COUNT(*) AS cnt
			FROM opportunities
			WHERE active = true AND is_latest = true
			  AND full_parent_path_name IS NOT NULL AND full_parent_path_name <> ''
			GROUP BY full_parent_path_name
		),
		expanded AS (
			SELECT
				array_to_string((string_to_array(path, '.'))[1:n], '.') AS prefix,
				cnt
			FROM path_counts
			CROSS JOIN LATERAL generate_series(1, array_length(string_to_array(path, '.'), 1)) AS n
		),
		counted AS (
			SELECT prefix, SUM(cnt)::int AS total_count
			FROM expanded
			GROUP BY prefix
		)
		INSERT INTO agencies (name, level, parent_path, opportunity_count)
		SELECT
			split_part(prefix, '.', array_length(string_to_array(prefix, '.'), 1)) AS name,
			CASE array_length(string_to_array(prefix, '.'), 1)
				WHEN 1 THEN 'department'
				WHEN 2 THEN 'sub_tier'
				ELSE 'command'
			END AS level,
			prefix AS parent_path,
			total_count
		FROM counted
		ON CONFLICT (parent_path) DO UPDATE SET
			opportunity_count = EXCLUDED.opportunity_count,
			name = EXCLUDED.name
	`)
	if err != nil {
		return 0, fmt.Errorf("refresh agencies: %w", err)
	}

	db.pool.Exec(ctx, `DELETE FROM agencies WHERE opportunity_count = 0`)

	return int(tag.RowsAffected()), nil
}

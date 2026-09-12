// Command backfill-awardee-name populates opportunities.awardee_name for rows
// ingested before the CSV path learned to parse it.
//
// SAM.gov's bulk CSV carries the awardee as a single column holding the company
// name and its full address run together. The API returns a clean name, but only
// enriches about a fifth of notices — and for roughly 9% of those it returns the
// same concatenated blob. The result was awardee_name populated on 21% of award
// notices and empty on the rest, even though the name was present all along.
//
// This applies parse.AwardeeNameOrBlob to existing rows. It is idempotent: a row
// whose awardee_name is already a clean name is left alone. Rows whose name has
// no recognisable legal suffix are skipped rather than guessed at, so a rerun
// after the parser improves will pick them up. A row whose stored name is itself
// an address with nothing to cut at is cleared, since an empty field is the
// intended outcome there and leaving the address in place is not.
//
// awardee_name is excluded from Opportunity.ContentHash, so this does not
// re-version the notices it touches.
//
//	DATABASE_URL=... go run ./cmd/backfill-awardee-name            # dry run
//	DATABASE_URL=... go run ./cmd/backfill-awardee-name -apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/parse"
	"github.com/jackc/pgx/v5/pgxpool"
)

const batchSize = 2000

type row struct {
	id      int64
	awardee string
	name    string
}

func main() {
	apply := flag.Bool("apply", false, "Write changes (default is a dry run)")
	limit := flag.Int("limit", 0, "Max rows to process (0 = all)")
	sample := flag.Int("sample", 10, "Print this many example changes")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	start := time.Now()
	var scanned, updated, skipped, unchanged, cleared int
	var printed int
	var lastID int64

	for {
		rows, err := fetch(ctx, pool, lastID, batchSize)
		if err != nil {
			log.Fatalf("fetch batch: %v", err)
		}
		if len(rows) == 0 {
			break
		}

		var ids []int64
		var names []string
		for _, r := range rows {
			scanned++
			lastID = r.id

			derived := parse.AwardeeNameOrBlob(r.name, r.awardee)
			switch {
			case derived == "" && r.name != "":
				// The stored name is unusable — an address with no legal suffix to
				// cut at — and nothing better can be derived. Clearing it is the
				// point: an empty field beats one holding a street address.
				cleared++
				ids = append(ids, r.id)
				names = append(names, "")
			case derived == "":
				skipped++ // no recognisable legal suffix — leave empty rather than guess
			case derived == r.name:
				unchanged++
			default:
				if printed < *sample {
					printed++
					from := r.name
					if from == "" {
						from = "(empty)"
					}
					fmt.Printf("  id=%-9d %s\n      from %q\n        to %q\n",
						r.id, truncate(r.awardee, 68), truncate(from, 60), derived)
				}
				ids = append(ids, r.id)
				names = append(names, derived)
				updated++
			}
		}

		if *apply && len(ids) > 0 {
			if err := update(ctx, pool, ids, names); err != nil {
				log.Fatalf("update batch: %v", err)
			}
		}

		if *limit > 0 && scanned >= *limit {
			break
		}
		if scanned%50000 < batchSize {
			log.Printf("progress: scanned=%d updated=%d cleared=%d skipped=%d", scanned, updated, cleared, skipped)
		}
	}

	mode := "DRY RUN — no changes written (pass -apply to write)"
	if *apply {
		mode = "APPLIED"
	}
	fmt.Printf("\n%s\n  scanned   %d\n  updated   %d\n  cleared   %d (stored value was an address)\n  unchanged %d\n  skipped   %d (no legal suffix)\n  elapsed   %s\n",
		mode, scanned, updated, cleared, unchanged, skipped, time.Since(start).Round(time.Millisecond))
}

// fetch pages by id so a long run stays stable while the pipeline writes.
func fetch(ctx context.Context, pool *pgxpool.Pool, afterID int64, n int) ([]row, error) {
	const q = `
		SELECT id, COALESCE(awardee, ''), COALESCE(awardee_name, '')
		FROM opportunities
		WHERE id > $1
		  AND (COALESCE(awardee, '') <> '' OR COALESCE(awardee_name, '') <> '')
		ORDER BY id
		LIMIT $2
	`
	rs, err := pool.Query(ctx, q, afterID, n)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var out []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.awardee, &r.name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rs.Err()
}

func update(ctx context.Context, pool *pgxpool.Pool, ids []int64, names []string) error {
	const q = `
		UPDATE opportunities AS o
		SET awardee_name = v.name
		FROM (SELECT unnest($1::bigint[]) AS id, unnest($2::text[]) AS name) AS v
		WHERE o.id = v.id
	`
	_, err := pool.Exec(ctx, q, ids, names)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

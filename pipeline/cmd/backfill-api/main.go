// Command backfill-api performs a one-time, API-only backfill of SAM.gov
// opportunities for a posted-date window the daily CSV pipeline missed (e.g.
// during a SAM.gov CSV outage). SAM.gov caps any single query at 1000
// retrievable results, so this slices each day by procurement type — keeping
// every query under the cap — and FAILS LOUD rather than silently truncating if
// a slice is still too large. It writes the raw records to pipeline.snap_api
// under a fresh snapshot-api ingestion run, then prints the reconcile
// invocation that upserts them API-only (no deactivation of CSV-only records).
//
// Defaults to a dry run. See README.md in this directory before running.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/samgov"
)

const dateLayout = "01/02/2006"

// ptypes is the full SAM.gov procurement-type set. Partitioning a day by these
// is exhaustive and non-overlapping (verified: per-day slice totals sum to the
// day total), and each slice has stayed well under the 1000 cap for the outage
// window. The >=1000 guard below catches any future day where that stops holding.
var ptypes = []string{"o", "p", "k", "r", "s", "g", "a", "i", "u"}

func main() {
	from := flag.String("from", "07/29/2026", "postedFrom (MM/DD/YYYY), inclusive")
	to := flag.String("to", time.Now().Format(dateLayout), "postedTo (MM/DD/YYYY), inclusive")
	write := flag.Bool("write", false, "actually write to the database (default false = dry run: fetch and report only)")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	dsn := os.Getenv("NEON_DATABASE_URL")
	apiKey := os.Getenv("SAM_API_KEY")
	if dsn == "" || apiKey == "" {
		fatal("NEON_DATABASE_URL and SAM_API_KEY must both be set in the environment")
	}

	fromT, err := time.Parse(dateLayout, *from)
	if err != nil {
		fatal("bad -from %q: %v", *from, err)
	}
	toT, err := time.Parse(dateLayout, *to)
	if err != nil {
		fatal("bad -to %q: %v", *to, err)
	}
	if toT.Before(fromT) {
		fatal("-to (%s) is before -from (%s)", *to, *from)
	}

	ctx := context.Background()
	db, err := database.New(ctx, dsn)
	if err != nil {
		fatal("connect to database: %v", err)
	}
	client := samgov.NewAPIClient(apiKey, logger, db)

	modeStr := "DRY RUN (no writes)"
	if *write {
		modeStr = "WRITE"
	}
	fmt.Printf("Backfill window %s .. %s  —  %s\n\n", *from, *to, modeStr)

	seen := make(map[string]bool)
	var rows []database.SnapAPIRow
	grandTotal := 0
	requests := 0

	for d := fromT; !d.After(toT); d = d.AddDate(0, 0, 1) {
		day := d.Format(dateLayout)
		dayTotal, dayNew := 0, 0
		for _, pt := range ptypes {
			resp, raw, err := client.FetchPagePtype(ctx, 0, 1000, day, day, pt)
			requests++
			if err != nil {
				fatal("fetch %s ptype=%s: %v", day, pt, err)
			}
			if resp.TotalRecords >= 1000 {
				fatal("SLICE TOO LARGE: %s ptype=%s reports %d records (>=1000 cap) — a finer sub-slice (e.g. by NAICS) is required; aborting to avoid silent truncation", day, pt, resp.TotalRecords)
			}
			dayTotal += resp.TotalRecords
			for i, r := range raw {
				nid := resp.OpportunitiesData[i].NoticeID
				if seen[nid] {
					continue
				}
				seen[nid] = true
				dayNew++
				rows = append(rows, database.SnapAPIRow{NoticeID: nid, RawData: r, ContentHash: hashRaw(r)})
			}
			time.Sleep(1 * time.Second) // deliberate pacing — polite, well under the daily cap
		}
		fmt.Printf("  %s  fetched=%-5d unique=%-5d\n", day, dayTotal, dayNew)
		grandTotal += dayTotal
	}

	fmt.Printf("\nSum of slice totals: %d\nUnique notice_ids:   %d\nAPI requests made:   %d\n", grandTotal, len(rows), requests)

	if !*write {
		fmt.Println("\nDRY RUN complete — nothing written. Re-run with -write to ingest.")
		return
	}
	if len(rows) == 0 {
		fmt.Println("\nNothing to ingest — no records in window.")
		return
	}

	snapshotDate := time.Now().UTC()
	runID, err := db.CreateIngestionRun(ctx, "snapshot-api", nil)
	if err != nil {
		fatal("create ingestion run: %v", err)
	}
	inserted, err := db.BulkInsertSnapAPI(ctx, runID, snapshotDate, rows)
	if err != nil {
		fatal("bulk insert snap_api (run %s): %v", runID, err)
	}
	if err := db.CompleteIngestionRun(ctx, runID, database.RunStats{Fetched: len(rows), Inserted: int(inserted)}); err != nil {
		fmt.Printf("warning: mark run complete: %v\n", err)
	}

	fmt.Printf("\n✅ Wrote %d snap_api rows under run_id %s\n", inserted, runID)
	execID := uuid.NewString()
	fmt.Printf("\nNEXT — upsert into the catalog (API-only reconcile; activeRunID stays nil so the\ndisappearance/mass-deactivation path is skipped). The execution_id makes reconcile record a\ncompleted 'reconcile' pipeline step, which advances the /api/status freshness marker and clears\nthe stale-data banner. Use an ASYNC invoke: a synchronous one can exceed the CLI's 60s read\ntimeout and get RETRIED, running reconcile concurrently.\n\n")
	fmt.Printf("  aws lambda invoke --function-name %s --profile govtrove --invocation-type Event \\\n    --cli-binary-format raw-in-base64-out \\\n    --payload '{\"execution_id\":\"%s\",\"api_result\":{\"status\":\"ok\",\"run_id\":\"%s\",\"job_type\":\"snapshot-api\"}}' /dev/stdout\n",
		reconcileFn(), execID, runID)
	fmt.Printf("\nIt returns 202 immediately and runs reconcile once (~2-3 min). Then verify:\n  SELECT count(*) FILTER (WHERE active) FROM opportunities WHERE is_latest=true;  -- should rise\n")
}

func hashRaw(d json.RawMessage) string {
	h := sha256.Sum256(d)
	return hex.EncodeToString(h[:])
}

func reconcileFn() string {
	if v := os.Getenv("RECONCILE_FUNCTION"); v != "" {
		return v
	}
	return "<reconcile fn — set RECONCILE_FUNCTION or resolve via: aws lambda list-functions --profile govtrove --query \"Functions[?contains(FunctionName,'reconcile')].FunctionName\">"
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

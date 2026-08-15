package searchtest_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/handriss/govtrove/api/internal/handlers"
	"github.com/handriss/govtrove/api/internal/repository"
)

var (
	server *httptest.Server
	pool   *pgxpool.Pool
	pgCtr  *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	var err error
	pgCtr, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("govtrove_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		slog.Error("failed to start postgres container", "error", err)
		os.Exit(1)
	}

	dbURL, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		slog.Error("failed to get connection string", "error", err)
		os.Exit(1)
	}

	pool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		slog.Error("failed to apply schema", "error", err)
		os.Exit(1)
	}

	if _, err := pool.Exec(ctx, seedSQL); err != nil {
		slog.Error("failed to seed data", "error", err)
		os.Exit(1)
	}

	oppRepo := repository.NewOpportunityRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	eventLog := handlers.NewEventLogger(eventRepo, logger)
	geoRepo := repository.NewGeoSynonymRepository(pool)
	oppHandler := handlers.NewOpportunityHandler(oppRepo, nil, logger, eventLog, userRepo, geoRepo)

	r := chi.NewRouter()
	r.Get("/api/opportunities", oppHandler.Search)
	r.Get("/api/opportunities/facets", oppHandler.GetFacets)

	server = httptest.NewServer(r)

	code := m.Run()

	server.Close()
	pool.Close()
	if pgCtr != nil {
		_ = pgCtr.Terminate(ctx)
	}
	os.Exit(code)
}

func TestSearch_KeywordSterilizer(t *testing.T) {
	params := url.Values{"q": {"sterilizer"}}
	result := searchGet(t, server.URL, params)

	// "sterilizer" stems to "steril" which matches title hits (TEST-001, TEST-006)
	// and also TEST-003's description containing "sterilization"
	if result.Total != 3 {
		t.Fatalf("expected 3 results, got %d: %v", result.Total, titles(result))
	}

	if !hasTitle(result, "Sterilizer Equipment Maintenance") {
		t.Error("missing 'Sterilizer Equipment Maintenance'")
	}
	if !hasTitle(result, "Sterilizer Supply Chain Analysis") {
		t.Error("missing 'Sterilizer Supply Chain Analysis'")
	}
	if !hasTitle(result, "Autoclave Repair and Calibration") {
		t.Error("missing 'Autoclave Repair and Calibration' (description match via stemming)")
	}
}

func TestSearch_FilterByType(t *testing.T) {
	params := url.Values{"type": {"o"}}
	result := searchGet(t, server.URL, params)

	if result.Total != 3 {
		t.Fatalf("expected 3 Solicitation results, got %d: %v", result.Total, titles(result))
	}

	for _, opp := range result.Opportunities {
		if opp.Type == nil || *opp.Type != "Solicitation" {
			t.Errorf("expected type 'Solicitation', got %v for %s", opp.Type, opp.Title)
		}
	}
}

func TestSearch_NoResults(t *testing.T) {
	params := url.Values{"q": {"xyznonexistent123"}}
	result := searchGet(t, server.URL, params)

	if result.Total != 0 {
		t.Fatalf("expected 0 results, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
	if result.TotalPages != 1 {
		t.Errorf("expected total_pages 1, got %d", result.TotalPages)
	}
	if len(result.Opportunities) != 0 {
		t.Errorf("expected empty opportunities slice, got %d items", len(result.Opportunities))
	}
}

// The cases below are taken verbatim from zero-result searches in search_events.

// "IT" is entirely stopwords, so phraseto_tsquery yields an empty tsquery that
// matches nothing. It used to veto the literal re-check it was ANDed with.
func TestSearch_QuotedStopwordOnlyPhrase(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {`"IT"`}})

	if result.Total == 0 {
		t.Fatalf(`searching "IT" returned 0; expected the IT support notice`)
	}
	if !hasTitle(result, "IT Support Desk Modernization") {
		t.Errorf("missing 'IT Support Desk Modernization', got %v", titles(result))
	}
}

// An undated notice is open, not expired. response_deadline >= x drops it
// because NULL >= x is NULL.
func TestSearch_ActiveOnlyKeepsUndatedNotices(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {"modernization"}, "active": {"true"}})

	if !hasTitle(result, "IT Support Desk Modernization") {
		t.Errorf("active=true dropped the undated notice; got %v", titles(result))
	}
}

func TestSearch_ActiveOnlyStillExcludesExpired(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {"painting"}, "active": {"true"}})

	if hasTitle(result, "Expired Facility Painting") {
		t.Errorf("active=true returned a closed notice: %v", titles(result))
	}
}

// Someone pasting a notice number wants that notice, even though it closed and
// its type sits outside the default filter.
func TestSearch_KnownItemSolicitationLookup(t *testing.T) {
	for _, q := range []string{"W519TC-25-D-A066", "W519TC25DA066", "w519tc25da066"} {
		params := url.Values{
			"q":      {q},
			"active": {"true"},
			"type":   {"Solicitation", "Presolicitation", "Sources Sought"},
		}
		result := searchGet(t, server.URL, params)
		if !hasTitle(result, "Range Instrumentation Support") {
			t.Errorf("known-item lookup %q returned %v; expected the notice regardless of filters", q, titles(result))
		}
	}
}

// A topic search must not be treated as a known-item lookup.
func TestSearch_KnownItemDoesNotHijackTopicSearch(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {"painting"}, "active": {"true"}})

	if hasTitle(result, "Range Instrumentation Support") {
		t.Errorf("topic search matched an unrelated notice number: %v", titles(result))
	}
}

// Quotes are an exact-substring match and can collapse thousands of hits to
// one; the response has to say so.
func TestSearch_QuoteCollapseOffersRelaxedQuery(t *testing.T) {
	quoted := searchGet(t, server.URL, url.Values{"q": {`"Utilization Management"`}})
	if quoted.Total != 1 {
		t.Fatalf("expected the quoted phrase to match exactly 1, got %d: %v", quoted.Total, titles(quoted))
	}
	if quoted.RelaxedQuery != "Utilization Management" {
		t.Errorf("expected relaxed_query %q, got %q", "Utilization Management", quoted.RelaxedQuery)
	}
	if quoted.RelaxedTotal <= quoted.Total {
		t.Errorf("expected relaxed_total > %d, got %d", quoted.Total, quoted.RelaxedTotal)
	}
}

func TestSearch_NoRelaxedHintWhenUnquoted(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {"Utilization Management"}})

	if result.RelaxedQuery != "" {
		t.Errorf("unquoted search should not offer a relaxed query, got %q", result.RelaxedQuery)
	}
}

// --- Coverage added 2026-08-15 for behaviours that shipped untested. ---

// The facets endpoint produces the headline result count from a SEPARATE code
// path with its own param serializer. When it drifted from search on
// 2026-08-14 the page showed "26 results" above a list of 12, and every unit
// test and API probe passed. Parity is the invariant worth pinning.
func TestFacets_TotalMatchesSearchTotal(t *testing.T) {
	cases := []url.Values{
		{"q": {"modernization"}, "active": {"true"}},
		{"q": {"painting"}, "active": {"true"}},
		{"q": {"sterilizer"}},
		{"active": {"true"}},
	}
	for _, params := range cases {
		s := searchGet(t, server.URL, params)
		f := facetsGet(t, server.URL, params)
		if s.Total != f.Total {
			t.Errorf("params %v: search total %d != facets total %d", params, s.Total, f.Total)
		}
	}
}

func TestFacets_HonoursActiveFilter(t *testing.T) {
	withActive := facetsGet(t, server.URL, url.Values{"q": {"painting"}, "active": {"true"}})
	without := facetsGet(t, server.URL, url.Values{"q": {"painting"}})

	if withActive.Total >= without.Total {
		t.Errorf("active=true must narrow the facet total: %d (active) vs %d (all)",
			withActive.Total, without.Total)
	}
}

// "Still open" and "deadline within a range" are deliberately different
// intents: only the former includes undated notices.
func TestSearch_ExplicitDeadlineRangeExcludesUndatedNotices(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{
		"q":             {"modernization"},
		"deadline_from": {"2020-01-01"},
	})

	if hasTitle(result, "IT Support Desk Modernization") {
		t.Errorf("an explicit deadline range must exclude the undated notice; got %v", titles(result))
	}
}

// Pins TODAY's behaviour: a phrase that is entirely stopwords falls back to a
// case-insensitive SUBSTRING match, so "IT" also matches Monitor and Unit and
// the pronoun "it". Moving to case-sensitive word-boundary matching should
// break this test — update it deliberately, don't delete it.
func TestSearch_QuotedStopwordPhraseMatchesSubstrings_CURRENT(t *testing.T) {
	result := searchGet(t, server.URL, url.Values{"q": {`"IT"`}})

	if !hasTitle(result, "Monitor Calibration Unit") {
		t.Errorf("expected substring semantics to match Monitor/Unit; got %v", titles(result))
	}
	if !hasTitle(result, "IT Support Desk Modernization") {
		t.Errorf("the genuine IT notice must match under any semantics; got %v", titles(result))
	}
}

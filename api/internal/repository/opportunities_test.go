package repository

import (
	"strings"
	"testing"

	"github.com/handriss/govtrove/api/internal/models"
)

// --- splitOR ---

func TestSplitOR_SingleTerm(t *testing.T) {
	got := splitOR("sterilizer")
	if len(got) != 1 || got[0] != "sterilizer" {
		t.Errorf("got %v", got)
	}
}

func TestSplitOR_TwoTerms(t *testing.T) {
	got := splitOR("sterilizer OR autoclave")
	if len(got) != 2 || got[0] != "sterilizer" || got[1] != "autoclave" {
		t.Errorf("got %v", got)
	}
}

func TestSplitOR_ThreeTerms(t *testing.T) {
	got := splitOR("a OR b OR c")
	if len(got) != 3 {
		t.Errorf("expected 3, got %v", got)
	}
}

func TestSplitOR_QuotedPhrasePreserved(t *testing.T) {
	got := splitOR(`"this OR that" OR sterilizer`)
	if len(got) != 2 {
		t.Fatalf("expected 2 segments, got %v", got)
	}
	if got[0] != `"this OR that"` {
		t.Errorf("segment 0: got %q", got[0])
	}
	if got[1] != "sterilizer" {
		t.Errorf("segment 1: got %q", got[1])
	}
}

func TestSplitOR_LowercaseOrNotSplit(t *testing.T) {
	got := splitOR("sterilizer or autoclave")
	if len(got) != 1 {
		t.Errorf("lowercase 'or' should not split, got %v", got)
	}
}

func TestSplitOR_OregonNotSplit(t *testing.T) {
	got := splitOR("Oregon")
	if len(got) != 1 || got[0] != "Oregon" {
		t.Errorf("got %v", got)
	}
}

func TestSplitOR_EmptySegmentsSkipped(t *testing.T) {
	got := splitOR("sterilizer OR  OR autoclave")
	if len(got) != 2 {
		t.Errorf("expected 2 (empty segment skipped), got %v", got)
	}
}

func TestSplitOR_TrailingOR(t *testing.T) {
	got := splitOR("sterilizer OR ")
	if len(got) != 1 || got[0] != "sterilizer" {
		t.Errorf("got %v", got)
	}
}

func TestSplitOR_LeadingOR(t *testing.T) {
	// " OR sterilizer" — leading space + OR + space
	got := splitOR(" OR sterilizer")
	if len(got) != 1 || got[0] != "sterilizer" {
		t.Errorf("got %v", got)
	}
}

// --- buildFilterConditions with OR ---

func TestBuildFilterConditions_SingleQuery(t *testing.T) {
	params := models.SearchParams{Query: "sterilizer", Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if ftsExpr == "" {
		t.Error("expected ftsExpr to be set")
	}
	if len(args) != 1 || args[0] != "sterilizer" {
		t.Errorf("args: %v", args)
	}
	// Should have active, is_latest, and FTS conditions
	if len(conditions) != 3 {
		t.Errorf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
}

func TestBuildFilterConditions_ORQuery(t *testing.T) {
	params := models.SearchParams{Query: "sterilizer OR autoclave", Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "sterilizer" || args[1] != "autoclave" {
		t.Errorf("args: %v", args)
	}
	if ftsExpr == "" {
		t.Error("expected ftsExpr to be set")
	}
	if !strings.Contains(ftsExpr, "||") {
		t.Errorf("ftsExpr should contain ||: %s", ftsExpr)
	}
	// active, is_latest, and the combined search_vector condition
	if len(conditions) != 3 {
		t.Errorf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	orCond := conditions[2]
	if !strings.Contains(orCond, "search_vector") || !strings.Contains(orCond, "||") {
		t.Errorf("expected search_vector condition with ||, got: %s", orCond)
	}
}

func TestBuildFilterConditions_ORWithQuotedPhrase(t *testing.T) {
	params := models.SearchParams{Query: `sterilizer OR "exact phrase" OR autoclave`, Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	// 2 websearch + 1 phraseto + 1 literal ILIKE re-check for the phrase = 4 args
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "sterilizer" {
		t.Errorf("args[0]: %v", args[0])
	}
	if args[1] != "exact phrase" {
		t.Errorf("args[1]: %v", args[1])
	}
	if args[2] != "%exact phrase%" {
		t.Errorf("args[2] should be the literal re-check: %v", args[2])
	}
	if args[3] != "autoclave" {
		t.Errorf("args[3]: %v", args[3])
	}
	if ftsExpr == "" || !strings.Contains(ftsExpr, "||") {
		t.Errorf("expected combined ftsExpr with ||, got: %s", ftsExpr)
	}
	// Should be a single search_vector condition combining all terms
	orCond := conditions[2]
	if !strings.Contains(orCond, "search_vector") || !strings.Contains(orCond, "phraseto_tsquery") {
		t.Errorf("expected search_vector with phraseto_tsquery, got: %s", orCond)
	}
}

func TestBuildFilterConditions_ORAllQuoted(t *testing.T) {
	params := models.SearchParams{Query: `"phrase one" OR "phrase two"`, Page: 1, Limit: 20}
	_, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if ftsExpr == "" || !strings.Contains(ftsExpr, "phraseto_tsquery") {
		t.Errorf("expected ftsExpr with phraseto_tsquery, got: %s", ftsExpr)
	}
	// each phrase contributes a phraseto_tsquery arg plus a literal ILIKE re-check
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "phrase one" || args[2] != "phrase two" {
		t.Errorf("args: %v", args)
	}
	if args[1] != "%phrase one%" || args[3] != "%phrase two%" {
		t.Errorf("expected literal re-checks: %v", args)
	}
}

// Spaces narrow. This is the regression that made a plain two-word search
// behave like OR and silently widen the result set.
func TestBuildFilterConditions_BareTermsAreANDed(t *testing.T) {
	params := models.SearchParams{Query: "cybersecurity training", Page: 1, Limit: 20}
	conditions, args, _, _ := buildFilterConditions(params, "", 1)

	if len(args) != 1 || args[0] != "cybersecurity training" {
		t.Fatalf("bare terms should pass through as one websearch_to_tsquery arg: %v", args)
	}
	cond := conditions[len(conditions)-1]
	if strings.Contains(cond, " OR ") {
		t.Errorf("a single segment must not produce an OR: %s", cond)
	}
}

// A quoted phrase next to a bare term must AND, not OR.
//
// The phrase and the remainder used to be combined into a single tsquery with
// `&&`. They are now separate ANDed conditions because an all-stopword phrase
// ("IT") produces an EMPTY tsquery, and an empty operand of `&&` matches
// nothing — which vetoed the whole segment. Ranking still combines them, so
// assert AND-ness on the condition and `&&` on the rank expression.
func TestBuildFilterConditions_PhrasePlusTermIsANDed(t *testing.T) {
	params := models.SearchParams{Query: `cybersecurity "zero trust"`, Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if len(args) != 3 {
		t.Fatalf("expected phrase + literal + remainder args, got %v", args)
	}
	cond := conditions[len(conditions)-1]
	if !strings.Contains(cond, " AND ") {
		t.Errorf("phrase and term must be ANDed: %s", cond)
	}
	if !strings.Contains(ftsExpr, "&&") {
		t.Errorf("rank expression must still combine phrase and term: %s", ftsExpr)
	}
	if !strings.Contains(cond, "ILIKE") {
		t.Errorf("quoted phrase should carry a literal re-check: %s", cond)
	}
	// Without this the empty-tsquery case can never match.
	if !strings.Contains(cond, "= ''::tsquery") {
		t.Errorf("phrase condition must tolerate a degenerate (all-stopword) tsquery: %s", cond)
	}
}

// Segments are still OR'd against each other; only the intra-segment shape changed.
func TestBuildFilterConditions_SegmentsAreORed(t *testing.T) {
	params := models.SearchParams{Query: `"zero trust" OR "supply chain"`, Page: 1, Limit: 20}
	conditions, _, _, ftsExpr := buildFilterConditions(params, "", 1)

	cond := conditions[len(conditions)-1]
	if !strings.Contains(cond, " OR ") {
		t.Errorf("two segments must be OR'd: %s", cond)
	}
	if !strings.Contains(ftsExpr, "||") {
		t.Errorf("rank expression should OR the segments: %s", ftsExpr)
	}
}

func TestBuildFilterConditions_NoQuery(t *testing.T) {
	params := models.SearchParams{Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if ftsExpr != "" {
		t.Errorf("expected empty ftsExpr, got: %s", ftsExpr)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
	if len(conditions) != 2 {
		t.Errorf("expected 2 conditions (active + is_latest), got %d", len(conditions))
	}
}

func TestBuildFilterConditions_GeoExpansion(t *testing.T) {
	params := models.SearchParams{
		Query:     "upper peninsula",
		GeoStates: []string{"MI"},
		Page:      1,
		Limit:     20,
	}
	conditions, args, _, _ := buildFilterConditions(params, "", 1)

	// active + is_latest + (FTS OR geo)
	if len(conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	combined := conditions[2]
	if !strings.Contains(combined, "search_vector") {
		t.Errorf("expected search_vector in combined condition, got: %s", combined)
	}
	if !strings.Contains(combined, "pop_state") {
		t.Errorf("expected pop_state in combined condition, got: %s", combined)
	}
	if !strings.Contains(combined, " OR ") {
		t.Errorf("expected OR in combined condition, got: %s", combined)
	}
	// args: FTS query + geo states
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
}

func TestBuildFilterConditions_GeoWithCities(t *testing.T) {
	params := models.SearchParams{
		Query:     "hampton roads",
		GeoStates: []string{"VA"},
		GeoCities: []string{"Norfolk", "Virginia Beach", "Newport News"},
		Page:      1,
		Limit:     20,
	}
	conditions, args, _, _ := buildFilterConditions(params, "", 1)

	combined := conditions[2]
	if !strings.Contains(combined, "pop_state") {
		t.Errorf("expected pop_state in condition, got: %s", combined)
	}
	if !strings.Contains(combined, "pop_city") {
		t.Errorf("expected pop_city in condition, got: %s", combined)
	}
	// args: FTS query + geo states + geo city patterns
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
}

func TestBuildFilterConditions_NoGeo(t *testing.T) {
	params := models.SearchParams{Query: "logistics", Page: 1, Limit: 20}
	conditions, args, _, _ := buildFilterConditions(params, "", 1)

	// No geo expansion: active + is_latest + FTS only
	if len(conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	if strings.Contains(conditions[2], "pop_state") {
		t.Errorf("unexpected pop_state in condition: %s", conditions[2])
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
	}
}

// --- Sort defaults. Untested until 2026-08-15 despite being changed twice. ---

// A typed query is a relevance request. This was the whole point of the
// 2026-08-14 change and nothing pinned it.
func TestBuildOrderClause_DefaultsToRelevanceWhenQueryPresent(t *testing.T) {
	r := &OpportunityRepository{}
	got := r.buildOrderClause(models.SearchParams{Query: "coffee"}, "websearch_to_tsquery('english', $1)")

	if !strings.Contains(got, "ts_rank") {
		t.Errorf("a typed query with no explicit sort must rank by relevance, got: %s", got)
	}
}

// Browsing with no query has nothing to rank, so newest-first is correct.
func TestBuildOrderClause_DefaultsToPostedDateWhenBrowsing(t *testing.T) {
	r := &OpportunityRepository{}
	got := r.buildOrderClause(models.SearchParams{}, "")

	if strings.Contains(got, "ts_rank") {
		t.Errorf("browsing must not rank by relevance, got: %s", got)
	}
	if !strings.Contains(got, "posted_date") {
		t.Errorf("browsing should sort by posted_date, got: %s", got)
	}
}

// relevance falls back safely when there is no FTS expression to rank against.
func TestBuildOrderClause_RelevanceWithoutQueryFallsBack(t *testing.T) {
	r := &OpportunityRepository{}
	got := r.buildOrderClause(models.SearchParams{Sort: "relevance"}, "")

	if strings.Contains(got, "ts_rank") {
		t.Errorf("ts_rank needs a tsquery; expected fallback, got: %s", got)
	}
}

// An explicit choice must survive, or the sort dropdown is decorative.
func TestBuildOrderClause_ExplicitSortWinsOverDefault(t *testing.T) {
	r := &OpportunityRepository{}
	got := r.buildOrderClause(models.SearchParams{Query: "coffee", Sort: "deadline"}, "x")

	if !strings.Contains(got, "response_deadline") {
		t.Errorf("explicit sort=deadline ignored, got: %s", got)
	}
	if strings.Contains(got, "ts_rank") {
		t.Errorf("explicit sort must not be overridden by the relevance default, got: %s", got)
	}
}

// --- Known-item detection. A false positive silently bypasses the open-only
// and notice-type filters, so the boundaries matter more than the happy path.

func TestLooksLikeSolicitationNumber(t *testing.T) {
	cases := []struct {
		q    string
		want bool
		why  string
	}{
		{"W15QKN26RA037", true, "plain notice number"},
		{"W15QKN-26-R-A037", true, "dashed notice number"},
		{"w519tc25da066", true, "lowercase still a notice number"},
		{"36C77626R0025", true, "VA-style number"},
		{"", false, "empty"},
		{"coffee", false, "no digits"},
		{"541611", false, "NAICS code: digits only, must stay a topic search"},
		{"424490", false, "NAICS code"},
		{"1550", false, "PSC code"},
		{"cloud migration", false, "contains a space"},
		{`"IT"`, false, "quoted phrase"},
		{"AB12", false, "too short to be a notice number"},
		{"IT services 2026", false, "multi-word"},
		{strings.Repeat("A1", 20), false, "too long"},
	}
	for _, c := range cases {
		if got := looksLikeSolicitationNumber(c.q); got != c.want {
			t.Errorf("looksLikeSolicitationNumber(%q) = %v, want %v (%s)", c.q, got, c.want, c.why)
		}
	}
}

func TestNormalizeSolNum(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"W15QKN-26-R-A037", "W15QKN26RA037"},
		{"w519tc 25 d a066", "W519TC25DA066"},
		{"36C776/26R0025", "36C77626R0025"},
	} {
		if got := normalizeSolNum(c.in); got != c.want {
			t.Errorf("normalizeSolNum(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

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

	// All go through FTS now: 2 websearch + 1 phraseto = 3 args
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "sterilizer" {
		t.Errorf("args[0]: %v", args[0])
	}
	if args[1] != "exact phrase" {
		t.Errorf("args[1]: %v", args[1])
	}
	if args[2] != "autoclave" {
		t.Errorf("args[2]: %v", args[2])
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
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "phrase one" || args[1] != "phrase two" {
		t.Errorf("args: %v", args)
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

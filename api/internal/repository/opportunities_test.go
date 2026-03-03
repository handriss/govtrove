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
	// active, is_latest, and the OR compound condition
	if len(conditions) != 3 {
		t.Errorf("expected 3 conditions, got %d: %v", len(conditions), conditions)
	}
	orCond := conditions[2]
	if !strings.HasPrefix(orCond, "(") || !strings.Contains(orCond, "||") {
		t.Errorf("expected compound condition with tsquery ||, got: %s", orCond)
	}
}

func TestBuildFilterConditions_ORWithQuotedPhrase(t *testing.T) {
	params := models.SearchParams{Query: `sterilizer OR "exact phrase" OR autoclave`, Page: 1, Limit: 20}
	conditions, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	// 2 FTS args + 1 ILIKE arg = 3
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "sterilizer" {
		t.Errorf("args[0]: %v", args[0])
	}
	if args[1] != "%exact phrase%" {
		t.Errorf("args[1]: %v", args[1])
	}
	if args[2] != "autoclave" {
		t.Errorf("args[2]: %v", args[2])
	}
	if ftsExpr == "" || !strings.Contains(ftsExpr, "||") {
		t.Errorf("expected combined ftsExpr with ||, got: %s", ftsExpr)
	}
	// The OR condition should contain both ILIKE and FTS parts
	orCond := conditions[2]
	if !strings.Contains(orCond, "ILIKE") || !strings.Contains(orCond, "search_vector") {
		t.Errorf("expected ILIKE + FTS in OR condition, got: %s", orCond)
	}
}

func TestBuildFilterConditions_ORAllQuoted(t *testing.T) {
	params := models.SearchParams{Query: `"phrase one" OR "phrase two"`, Page: 1, Limit: 20}
	_, args, _, ftsExpr := buildFilterConditions(params, "", 1)

	if ftsExpr != "" {
		t.Errorf("expected empty ftsExpr (all quoted), got: %s", ftsExpr)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "%phrase one%" || args[1] != "%phrase two%" {
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

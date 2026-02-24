package main

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestDiffResourceLinks_BothNil(t *testing.T) {
	added, removed := diffResourceLinks(nil, nil)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_OldNilNewHasLinks(t *testing.T) {
	added, removed := diffResourceLinks(nil, strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`))
	if len(added) != 2 {
		t.Errorf("expected 2 added, got %d: %v", len(added), added)
	}
	if len(removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(removed))
	}
}

func TestDiffResourceLinks_OldHasLinksNewNil(t *testing.T) {
	added, removed := diffResourceLinks(strPtr(`["https://example.com/a.pdf"]`), nil)
	if len(added) != 0 {
		t.Errorf("expected 0 added, got %d", len(added))
	}
	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d: %v", len(removed), removed)
	}
	if len(removed) == 1 && removed[0] != "https://example.com/a.pdf" {
		t.Errorf("wrong removed link: %v", removed)
	}
}

func TestDiffResourceLinks_SameLinks(t *testing.T) {
	links := strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`)
	added, removed := diffResourceLinks(links, links)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff for identical links, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_SameLinksDifferentOrder(t *testing.T) {
	old := strPtr(`["https://example.com/b.pdf","https://example.com/a.pdf"]`)
	new := strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff for reordered links, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_LinksAdded(t *testing.T) {
	old := strPtr(`["https://example.com/a.pdf"]`)
	new := strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 1 || added[0] != "https://example.com/b.pdf" {
		t.Errorf("expected b.pdf added, got %v", added)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removed, got %v", removed)
	}
}

func TestDiffResourceLinks_LinksRemoved(t *testing.T) {
	old := strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`)
	new := strPtr(`["https://example.com/a.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
	if len(removed) != 1 || removed[0] != "https://example.com/b.pdf" {
		t.Errorf("expected b.pdf removed, got %v", removed)
	}
}

func TestDiffResourceLinks_AddedAndRemoved(t *testing.T) {
	old := strPtr(`["https://example.com/a.pdf","https://example.com/b.pdf"]`)
	new := strPtr(`["https://example.com/b.pdf","https://example.com/c.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 1 || added[0] != "https://example.com/c.pdf" {
		t.Errorf("expected c.pdf added, got %v", added)
	}
	if len(removed) != 1 || removed[0] != "https://example.com/a.pdf" {
		t.Errorf("expected a.pdf removed, got %v", removed)
	}
}

func TestDiffResourceLinks_EmptyArrayVsNil(t *testing.T) {
	added, removed := diffResourceLinks(strPtr(`[]`), nil)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff for empty array vs nil, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_NilVsEmptyArray(t *testing.T) {
	added, removed := diffResourceLinks(nil, strPtr(`[]`))
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff for nil vs empty array, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_MalformedJSON(t *testing.T) {
	// Malformed JSON should degrade gracefully — treated as empty
	added, removed := diffResourceLinks(strPtr(`not json`), strPtr(`["https://example.com/a.pdf"]`))
	if len(added) != 1 {
		t.Errorf("expected 1 added (old malformed treated as empty), got %d", len(added))
	}
	if len(removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(removed))
	}
}

func TestDiffResourceLinks_BothMalformedJSON(t *testing.T) {
	added, removed := diffResourceLinks(strPtr(`{bad`), strPtr(`{bad`))
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff for both malformed, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_CompleteReplacement(t *testing.T) {
	old := strPtr(`["https://example.com/old1.pdf","https://example.com/old2.pdf"]`)
	new := strPtr(`["https://example.com/new1.pdf","https://example.com/new2.pdf","https://example.com/new3.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 3 {
		t.Errorf("expected 3 added, got %d: %v", len(added), added)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 removed, got %d: %v", len(removed), removed)
	}
}

func TestDiffResourceLinks_DuplicatesInArray(t *testing.T) {
	// Duplicates collapse in set comparison — this is acceptable behavior
	old := strPtr(`["https://example.com/a.pdf","https://example.com/a.pdf"]`)
	new := strPtr(`["https://example.com/a.pdf"]`)
	added, removed := diffResourceLinks(old, new)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected no diff when duplicates collapse, got added=%v removed=%v", added, removed)
	}
}

func TestDiffResourceLinks_SingleLinkToEmpty(t *testing.T) {
	added, removed := diffResourceLinks(strPtr(`["https://example.com/only.pdf"]`), strPtr(`[]`))
	if len(removed) != 1 || removed[0] != "https://example.com/only.pdf" {
		t.Errorf("expected only.pdf removed, got %v", removed)
	}
	if len(added) != 0 {
		t.Errorf("expected no added, got %v", added)
	}
}

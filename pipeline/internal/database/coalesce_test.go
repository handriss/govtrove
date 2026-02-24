package database

import (
	"testing"

	"github.com/handriss/govtrove/pipeline/internal/reconcile"
)

func TestCoalesceOpp_AwardeePreservesCSVValue(t *testing.T) {
	// CSV awardee is richer (has address), API only has company name.
	// coalesceOpp should NOT overwrite the CSV value.
	existing := reconcile.Opportunity{
		NoticeID: "N001",
		Awardee:  "ACME CORP Springfield IL 62701 USA",
	}
	api := reconcile.Opportunity{
		NoticeID: "N001",
		Awardee:  "ACME CORP",
	}
	merged := coalesceOpp(existing, api)
	if merged.Awardee != "ACME CORP Springfield IL 62701 USA" {
		t.Errorf("expected CSV awardee preserved, got %q", merged.Awardee)
	}
}

func TestCoalesceOpp_AwardeeFillsEmpty(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID: "N002",
		Awardee:  "",
	}
	api := reconcile.Opportunity{
		NoticeID: "N002",
		Awardee:  "ACME CORP",
	}
	merged := coalesceOpp(existing, api)
	if merged.Awardee != "ACME CORP" {
		t.Errorf("expected API awardee to fill empty, got %q", merged.Awardee)
	}
}

func TestCoalesceOpp_AwardeeBothEmpty(t *testing.T) {
	existing := reconcile.Opportunity{NoticeID: "N003"}
	api := reconcile.Opportunity{NoticeID: "N003"}
	merged := coalesceOpp(existing, api)
	if merged.Awardee != "" {
		t.Errorf("expected empty awardee, got %q", merged.Awardee)
	}
}

func TestCoalesceOpp_AwardeeNameFromAPI(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:    "N004",
		AwardeeName: "Old Name",
	}
	api := reconcile.Opportunity{
		NoticeID:    "N004",
		AwardeeName: "New Name",
	}
	merged := coalesceOpp(existing, api)
	if merged.AwardeeName != "New Name" {
		t.Errorf("expected API AwardeeName to win, got %q", merged.AwardeeName)
	}
}

func TestCoalesceOpp_AwardeeNameFillsEmpty(t *testing.T) {
	existing := reconcile.Opportunity{NoticeID: "N005"}
	api := reconcile.Opportunity{
		NoticeID:    "N005",
		AwardeeName: "ACME Corp",
	}
	merged := coalesceOpp(existing, api)
	if merged.AwardeeName != "ACME Corp" {
		t.Errorf("expected AwardeeName filled, got %q", merged.AwardeeName)
	}
}

func TestCoalesceOpp_AwardeeNamePreservedWhenAPIEmpty(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:    "N006",
		AwardeeName: "Existing Name",
	}
	api := reconcile.Opportunity{
		NoticeID:    "N006",
		AwardeeName: "",
	}
	merged := coalesceOpp(existing, api)
	if merged.AwardeeName != "Existing Name" {
		t.Errorf("expected existing AwardeeName preserved, got %q", merged.AwardeeName)
	}
}

func TestCoalesceOpp_PopCodesFilled(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID: "N007",
		PopCity:  "Springfield",
		PopState: "Illinois",
	}
	api := reconcile.Opportunity{
		NoticeID:       "N007",
		PopCityCode:    "12345",
		PopStateCode:   "IL",
		PopCountryCode: "USA",
	}
	merged := coalesceOpp(existing, api)

	if merged.PopCityCode != "12345" {
		t.Errorf("expected PopCityCode 12345, got %q", merged.PopCityCode)
	}
	if merged.PopStateCode != "IL" {
		t.Errorf("expected PopStateCode IL, got %q", merged.PopStateCode)
	}
	if merged.PopCountryCode != "USA" {
		t.Errorf("expected PopCountryCode USA, got %q", merged.PopCountryCode)
	}
	// Existing name fields preserved
	if merged.PopCity != "Springfield" {
		t.Errorf("expected PopCity preserved, got %q", merged.PopCity)
	}
}

func TestCoalesceOpp_PopCodesPreservedWhenAPIEmpty(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:       "N008",
		PopCityCode:    "12345",
		PopStateCode:   "IL",
		PopCountryCode: "USA",
	}
	api := reconcile.Opportunity{NoticeID: "N008"}
	merged := coalesceOpp(existing, api)

	if merged.PopCityCode != "12345" {
		t.Errorf("expected existing PopCityCode preserved, got %q", merged.PopCityCode)
	}
	if merged.PopStateCode != "IL" {
		t.Errorf("expected existing PopStateCode preserved, got %q", merged.PopStateCode)
	}
	if merged.PopCountryCode != "USA" {
		t.Errorf("expected existing PopCountryCode preserved, got %q", merged.PopCountryCode)
	}
}

func TestCoalesceOpp_PopCodesOverwritten(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:     "N009",
		PopStateCode: "OLD",
	}
	api := reconcile.Opportunity{
		NoticeID:     "N009",
		PopStateCode: "NEW",
	}
	merged := coalesceOpp(existing, api)
	if merged.PopStateCode != "NEW" {
		t.Errorf("expected PopStateCode overwritten, got %q", merged.PopStateCode)
	}
}

func TestCoalesceOpp_ResourceLinksFromAPI(t *testing.T) {
	existing := reconcile.Opportunity{NoticeID: "N010"}
	api := reconcile.Opportunity{
		NoticeID:      "N010",
		ResourceLinks: []string{"https://example.com/a.pdf"},
	}
	merged := coalesceOpp(existing, api)
	if len(merged.ResourceLinks) != 1 || merged.ResourceLinks[0] != "https://example.com/a.pdf" {
		t.Errorf("expected API resource links, got %v", merged.ResourceLinks)
	}
}

func TestCoalesceOpp_ResourceLinksPreservedWhenAPIEmpty(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:      "N011",
		ResourceLinks: []string{"https://example.com/existing.pdf"},
	}
	api := reconcile.Opportunity{NoticeID: "N011"}
	merged := coalesceOpp(existing, api)
	if len(merged.ResourceLinks) != 1 || merged.ResourceLinks[0] != "https://example.com/existing.pdf" {
		t.Errorf("expected existing resource links preserved, got %v", merged.ResourceLinks)
	}
}

func TestCoalesceOpp_ResourceLinksReplacedByAPI(t *testing.T) {
	existing := reconcile.Opportunity{
		NoticeID:      "N012",
		ResourceLinks: []string{"https://example.com/old.pdf"},
	}
	api := reconcile.Opportunity{
		NoticeID:      "N012",
		ResourceLinks: []string{"https://example.com/new.pdf", "https://example.com/new2.pdf"},
	}
	merged := coalesceOpp(existing, api)
	if len(merged.ResourceLinks) != 2 {
		t.Errorf("expected 2 resource links, got %d", len(merged.ResourceLinks))
	}
}

func TestCoalesceOpp_AwardeeAndAwardeeNameIndependent(t *testing.T) {
	// CSV record has rich Awardee, API adds AwardeeName without touching Awardee
	existing := reconcile.Opportunity{
		NoticeID: "N013",
		Awardee:  "BIG CORP Washington DC 20001 USA",
	}
	api := reconcile.Opportunity{
		NoticeID:    "N013",
		AwardeeName: "BIG CORP",
		Awardee:     "BIG CORP",
	}
	merged := coalesceOpp(existing, api)
	if merged.Awardee != "BIG CORP Washington DC 20001 USA" {
		t.Errorf("expected CSV awardee preserved when existing non-empty, got %q", merged.Awardee)
	}
	if merged.AwardeeName != "BIG CORP" {
		t.Errorf("expected AwardeeName set from API, got %q", merged.AwardeeName)
	}
}

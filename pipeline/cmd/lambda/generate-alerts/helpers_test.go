package main

import (
	"testing"
	"time"
)

// --- deadlinePresetToDate ---

func TestDeadlinePresetToDate_7Days(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("7", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("7 days: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_14Days(t *testing.T) {
	now := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("14", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("14 days: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_30Days(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("30", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := now.AddDate(0, 0, 30)
	if !got.Equal(want) {
		t.Errorf("30 days: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_60Days(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("60", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := now.AddDate(0, 0, 60)
	if !got.Equal(want) {
		t.Errorf("60 days: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_90Days(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("90", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := now.AddDate(0, 0, 90)
	if !got.Equal(want) {
		t.Errorf("90 days: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_QuarterQ1(t *testing.T) {
	// Jan 15 → end of Q1 = March 31
	now := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("quarter", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	// Q1 end month = ((1-1)/3+1)*3+1 = 4, Day=0 → March 31
	want := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("quarter Q1: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_QuarterQ2(t *testing.T) {
	// May 1 → end of Q2 = June 30
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("quarter", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("quarter Q2: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_QuarterQ3(t *testing.T) {
	// Sep 15 → end of Q3 = Sep 30
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("quarter", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("quarter Q3: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_QuarterQ4(t *testing.T) {
	// Nov 1 → end of Q4 = Dec 31
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	got := deadlinePresetToDate("quarter", now)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("quarter Q4: got %v, want %v", *got, want)
	}
}

func TestDeadlinePresetToDate_Unknown(t *testing.T) {
	now := time.Now()
	got := deadlinePresetToDate("unknown", now)
	if got != nil {
		t.Errorf("expected nil for unknown preset, got %v", *got)
	}
}

func TestDeadlinePresetToDate_Empty(t *testing.T) {
	now := time.Now()
	got := deadlinePresetToDate("", now)
	if got != nil {
		t.Errorf("expected nil for empty preset, got %v", *got)
	}
}

// --- friendlyFieldName ---

func TestFriendlyFieldName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"response_deadline", "Response Deadline"},
		{"archive_date", "Archive Date"},
		{"description", "Description"},
		{"set_aside_code", "Set-Aside"},
		{"award_amount", "Award Amount"},
		{"some_other_field", "some_other_field"},
	}
	for _, c := range cases {
		got := friendlyFieldName(c.input)
		if got != c.want {
			t.Errorf("friendlyFieldName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// --- strPtrEqual ---

func TestStrPtrEqual_BothNil(t *testing.T) {
	if !strPtrEqual(nil, nil) {
		t.Error("expected equal for both nil")
	}
}

func TestStrPtrEqual_OneNil(t *testing.T) {
	s := "hello"
	if strPtrEqual(&s, nil) {
		t.Error("expected not equal when only left is non-nil")
	}
	if strPtrEqual(nil, &s) {
		t.Error("expected not equal when only right is non-nil")
	}
}

func TestStrPtrEqual_Same(t *testing.T) {
	a, b := "same", "same"
	if !strPtrEqual(&a, &b) {
		t.Error("expected equal for same values")
	}
}

func TestStrPtrEqual_Different(t *testing.T) {
	a, b := "foo", "bar"
	if strPtrEqual(&a, &b) {
		t.Error("expected not equal for different values")
	}
}

// --- timePtrEqual ---

func TestTimePtrEqual_BothNil(t *testing.T) {
	if !timePtrEqual(nil, nil) {
		t.Error("expected equal for both nil")
	}
}

func TestTimePtrEqual_OneNil(t *testing.T) {
	now := time.Now()
	if timePtrEqual(&now, nil) {
		t.Error("expected not equal")
	}
	if timePtrEqual(nil, &now) {
		t.Error("expected not equal")
	}
}

func TestTimePtrEqual_Same(t *testing.T) {
	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !timePtrEqual(&a, &b) {
		t.Error("expected equal for same time")
	}
}

func TestTimePtrEqual_Different(t *testing.T) {
	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if timePtrEqual(&a, &b) {
		t.Error("expected not equal for different times")
	}
}

// --- floatPtrEqual ---

func TestFloatPtrEqual_BothNil(t *testing.T) {
	if !floatPtrEqual(nil, nil) {
		t.Error("expected equal for both nil")
	}
}

func TestFloatPtrEqual_OneNil(t *testing.T) {
	f := 1.0
	if floatPtrEqual(&f, nil) {
		t.Error("expected not equal")
	}
}

func TestFloatPtrEqual_Same(t *testing.T) {
	a, b := 99.99, 99.99
	if !floatPtrEqual(&a, &b) {
		t.Error("expected equal")
	}
}

func TestFloatPtrEqual_Different(t *testing.T) {
	a, b := 1.0, 2.0
	if floatPtrEqual(&a, &b) {
		t.Error("expected not equal")
	}
}

// --- diffPtrs ---

func TestDiffPtrs_BothPopulated(t *testing.T) {
	old, new := "old_val", "new_val"
	d := diffPtrs("title", &old, &new)
	if d["field"] != "title" {
		t.Errorf("field: got %v", d["field"])
	}
	if d["old"] != "old_val" {
		t.Errorf("old: got %v", d["old"])
	}
	if d["new"] != "new_val" {
		t.Errorf("new: got %v", d["new"])
	}
}

func TestDiffPtrs_OldNil(t *testing.T) {
	new := "new_val"
	d := diffPtrs("title", nil, &new)
	if _, ok := d["old"]; ok {
		t.Error("expected no 'old' key when old is nil")
	}
	if d["new"] != "new_val" {
		t.Errorf("new: got %v", d["new"])
	}
}

func TestDiffPtrs_NewNil(t *testing.T) {
	old := "old_val"
	d := diffPtrs("title", &old, nil)
	if d["old"] != "old_val" {
		t.Errorf("old: got %v", d["old"])
	}
	if _, ok := d["new"]; ok {
		t.Error("expected no 'new' key when new is nil")
	}
}

// --- diffTimePtrs ---

func TestDiffTimePtrs_BothPopulated(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	d := diffTimePtrs("response_deadline", &old, &new)
	if d["field"] != "response_deadline" {
		t.Errorf("field: got %v", d["field"])
	}
	if d["old"] != "2026-01-01T00:00:00Z" {
		t.Errorf("old: got %v", d["old"])
	}
	if d["new"] != "2026-06-01T00:00:00Z" {
		t.Errorf("new: got %v", d["new"])
	}
}

func TestDiffTimePtrs_OldNil(t *testing.T) {
	new := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	d := diffTimePtrs("archive_date", nil, &new)
	if _, ok := d["old"]; ok {
		t.Error("expected no 'old' key")
	}
	if d["new"] != "2026-06-01T00:00:00Z" {
		t.Errorf("new: got %v", d["new"])
	}
}

// --- diffFloatPtrs ---

func TestDiffFloatPtrs_BothPopulated(t *testing.T) {
	old, new := 1000.50, 2000.75
	d := diffFloatPtrs("award_amount", &old, &new)
	if d["field"] != "award_amount" {
		t.Errorf("field: got %v", d["field"])
	}
	if d["old"] != "1000.50" {
		t.Errorf("old: got %v", d["old"])
	}
	if d["new"] != "2000.75" {
		t.Errorf("new: got %v", d["new"])
	}
}

func TestDiffFloatPtrs_NewNil(t *testing.T) {
	old := 1000.50
	d := diffFloatPtrs("award_amount", &old, nil)
	if d["old"] != "1000.50" {
		t.Errorf("old: got %v", d["old"])
	}
	if _, ok := d["new"]; ok {
		t.Error("expected no 'new' key")
	}
}

// --- appendFilterConditions ---

func TestAppendFilterConditions_Empty(t *testing.T) {
	conditions := []string{"active = true"}
	args := []any{}
	f := savedFilters{}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 1 {
		t.Errorf("expected 1 condition, got %d: %v", len(got), got)
	}
	if len(gotArgs) != 0 {
		t.Errorf("expected 0 args, got %d", len(gotArgs))
	}
	if argNum != 1 {
		t.Errorf("expected argNum 1, got %d", argNum)
	}
}

func TestAppendFilterConditions_Keyword(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{Keyword: "cybersecurity"}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 1 {
		t.Errorf("expected 1 condition, got %d", len(got))
	}
	if got[0] != "search_vector @@ websearch_to_tsquery('english', $1)" {
		t.Errorf("unexpected condition: %s", got[0])
	}
	if len(gotArgs) != 1 || gotArgs[0] != "cybersecurity" {
		t.Errorf("unexpected args: %v", gotArgs)
	}
	if argNum != 2 {
		t.Errorf("expected argNum 2, got %d", argNum)
	}
}

func TestAppendFilterConditions_NoticeType(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{NoticeType: []string{"o", "p"}}

	got, gotArgs, _ := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 1 {
		t.Errorf("expected 1 condition, got %d", len(got))
	}
	if got[0] != "type = ANY($1)" {
		t.Errorf("unexpected condition: %s", got[0])
	}
	types := gotArgs[0].([]string)
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}
}

func TestAppendFilterConditions_SetAside(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{SetAside: []string{"SBA", "8A"}}

	got, _, _ := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 1 {
		t.Errorf("expected 1 condition, got %d", len(got))
	}
	if got[0] != "set_aside_code = ANY($1)" {
		t.Errorf("unexpected condition: %s", got[0])
	}
}

func TestAppendFilterConditions_NAICS(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{NAICS: []string{"541511"}}

	got, _, _ := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 1 {
		t.Errorf("expected 1 condition, got %d", len(got))
	}
	if got[0] != "naics_code = ANY($1)" {
		t.Errorf("unexpected condition: %s", got[0])
	}
}

func TestAppendFilterConditions_PSC(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{PSC: []string{"D302"}}

	got, _, _ := appendFilterConditions(conditions, args, 1, f)
	if got[0] != "classification_code = ANY($1)" {
		t.Errorf("unexpected condition: %s", got[0])
	}
}

func TestAppendFilterConditions_Department(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{Department: "Defense"}

	got, gotArgs, _ := appendFilterConditions(conditions, args, 1, f)
	if got[0] != "department ILIKE $1" {
		t.Errorf("unexpected condition: %s", got[0])
	}
	if gotArgs[0] != "%Defense%" {
		t.Errorf("expected ILIKE wrapping, got %v", gotArgs[0])
	}
}

func TestAppendFilterConditions_State(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{State: "Virginia"}

	got, gotArgs, _ := appendFilterConditions(conditions, args, 1, f)
	if got[0] != "pop_state = $1" {
		t.Errorf("unexpected condition: %s", got[0])
	}
	if gotArgs[0] != "Virginia" {
		t.Errorf("expected Virginia, got %v", gotArgs[0])
	}
}

func TestAppendFilterConditions_PostedDateRange(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{PostedFrom: "2026-01-01", PostedTo: "2026-06-30"}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 2 {
		t.Errorf("expected 2 conditions, got %d: %v", len(got), got)
	}
	if got[0] != "posted_date >= $1" {
		t.Errorf("unexpected from condition: %s", got[0])
	}
	if got[1] != "posted_date <= $2" {
		t.Errorf("unexpected to condition: %s", got[1])
	}
	if len(gotArgs) != 2 {
		t.Errorf("expected 2 args, got %d", len(gotArgs))
	}
	if argNum != 3 {
		t.Errorf("expected argNum 3, got %d", argNum)
	}
}

func TestAppendFilterConditions_PostedDateInvalid(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{PostedFrom: "not-a-date", PostedTo: "also-bad"}

	got, gotArgs, _ := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 0 {
		t.Errorf("expected 0 conditions for invalid dates, got %d: %v", len(got), got)
	}
	if len(gotArgs) != 0 {
		t.Errorf("expected 0 args, got %d", len(gotArgs))
	}
}

func TestAppendFilterConditions_DeadlinePreset(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{DeadlinePreset: "30"}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 1, f)
	// Should add >= today AND <= today+30
	if len(got) != 2 {
		t.Errorf("expected 2 conditions, got %d: %v", len(got), got)
	}
	if got[0] != "response_deadline >= $1" {
		t.Errorf("unexpected from condition: %s", got[0])
	}
	if got[1] != "response_deadline <= $2" {
		t.Errorf("unexpected to condition: %s", got[1])
	}
	if len(gotArgs) != 2 {
		t.Errorf("expected 2 args, got %d", len(gotArgs))
	}
	if argNum != 3 {
		t.Errorf("expected argNum 3, got %d", argNum)
	}
}

func TestAppendFilterConditions_DeadlinePresetUnknown(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{DeadlinePreset: "unknown_preset"}

	got, gotArgs, _ := appendFilterConditions(conditions, args, 1, f)
	// Should add >= today (always) but no upper bound (unknown preset → nil)
	if len(got) != 1 {
		t.Errorf("expected 1 condition (lower bound only), got %d: %v", len(got), got)
	}
	if len(gotArgs) != 1 {
		t.Errorf("expected 1 arg, got %d", len(gotArgs))
	}
}

func TestAppendFilterConditions_DeadlineDateRange(t *testing.T) {
	conditions := []string{}
	args := []any{}
	f := savedFilters{DeadlineFrom: "2026-03-01", DeadlineTo: "2026-06-01"}

	got, _, argNum := appendFilterConditions(conditions, args, 1, f)
	if len(got) != 2 {
		t.Errorf("expected 2 conditions, got %d: %v", len(got), got)
	}
	if argNum != 3 {
		t.Errorf("expected argNum 3, got %d", argNum)
	}
}

func TestAppendFilterConditions_DeadlinePresetOverridesRange(t *testing.T) {
	conditions := []string{}
	args := []any{}
	// When DeadlinePreset is set, DeadlineFrom/To should be ignored
	f := savedFilters{
		DeadlinePreset: "14",
		DeadlineFrom:   "2020-01-01",
		DeadlineTo:     "2030-12-31",
	}

	got, _, _ := appendFilterConditions(conditions, args, 1, f)
	// Should have 2 conditions from preset (>= today, <= today+14), NOT from the date range
	if len(got) != 2 {
		t.Errorf("expected 2 conditions from preset, got %d: %v", len(got), got)
	}
	if got[0] != "response_deadline >= $1" {
		t.Errorf("unexpected condition: %s", got[0])
	}
}

func TestAppendFilterConditions_AllFilters(t *testing.T) {
	conditions := []string{"active = true"}
	args := []any{}
	f := savedFilters{
		Keyword:    "cloud",
		NAICS:      []string{"541511"},
		PSC:        []string{"D302"},
		SetAside:   []string{"SBA"},
		Department: "Defense",
		State:      "Virginia",
		NoticeType: []string{"o"},
		PostedFrom: "2026-01-01",
		PostedTo:   "2026-12-31",
	}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 1, f)
	// 1 base + keyword + noticeType + setAside + naics + psc + department + state + postedFrom + postedTo = 10
	if len(got) != 10 {
		t.Errorf("expected 10 conditions, got %d: %v", len(got), got)
	}
	if len(gotArgs) != 9 {
		t.Errorf("expected 9 args, got %d", len(gotArgs))
	}
	if argNum != 10 {
		t.Errorf("expected argNum 10, got %d", argNum)
	}
}

func TestAppendFilterConditions_ArgNumContinuation(t *testing.T) {
	// Verify argNum is correctly continued from a previous value
	conditions := []string{"created_at > $1"}
	args := []any{time.Now()}
	f := savedFilters{Keyword: "test", State: "Texas"}

	got, gotArgs, argNum := appendFilterConditions(conditions, args, 2, f)
	if len(got) != 3 {
		t.Errorf("expected 3 conditions, got %d", len(got))
	}
	if got[1] != "search_vector @@ websearch_to_tsquery('english', $2)" {
		t.Errorf("unexpected keyword condition: %s", got[1])
	}
	if got[2] != "pop_state = $3" {
		t.Errorf("unexpected state condition: %s", got[2])
	}
	if len(gotArgs) != 3 {
		t.Errorf("expected 3 args, got %d", len(gotArgs))
	}
	if argNum != 4 {
		t.Errorf("expected argNum 4, got %d", argNum)
	}
}

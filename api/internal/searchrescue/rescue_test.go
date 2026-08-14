package searchrescue

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

type fakeProber struct {
	countFn func(p models.SearchParams) int
	calls   int
}

func (f *fakeProber) Count(_ context.Context, p models.SearchParams) (int, error) {
	f.calls++
	return f.countFn(p), nil
}

var fixedNow = time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

func newTestService(prober Prober, llm *LLMClient) *Service {
	s := New(prober, llm, nil, slog.Default())
	s.now = func() time.Time { return fixedNow }
	return s
}

func date(s string) *time.Time {
	t, err := time.Parse(dateFmt, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestSignificant(t *testing.T) {
	cases := []struct {
		orig, n int
		want    bool
	}{
		{0, 0, false},
		{0, 1, true},
		{0, 500, true},
		{2, 5, false},  // 3x but under the 10 floor
		{2, 10, true},  // meets both the floor and 3x
		{4, 11, false}, // under 3x
		{4, 12, true},
		{20, 59, false}, // under 3x
		{20, 60, true},
	}
	for _, c := range cases {
		if got := significant(c.orig, c.n); got != c.want {
			t.Errorf("significant(%d, %d) = %v, want %v", c.orig, c.n, got, c.want)
		}
	}
}

func TestParseQuery(t *testing.T) {
	pq := parseQuery(`janitorial "IT services" OR cleaning -award`)
	if len(pq.phrases) != 1 || pq.phrases[0] != "IT services" {
		t.Errorf("phrases = %v", pq.phrases)
	}
	if len(pq.terms) != 2 || pq.terms[0] != "janitorial" || pq.terms[1] != "cleaning" {
		t.Errorf("terms = %v", pq.terms)
	}
	if !pq.hasOR {
		t.Error("expected hasOR")
	}
	if len(pq.excluded) != 1 || pq.excluded[0] != "award" {
		t.Errorf("excluded = %v", pq.excluded)
	}
}

func TestStaticCandidates(t *testing.T) {
	today := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)

	t.Run("inverted deadline range", func(t *testing.T) {
		p := models.SearchParams{DeadlineFrom: date("2026-09-01"), DeadlineTo: date("2026-08-10")}
		cs := staticCandidates(p, parseQuery(""), "", today)
		if len(cs) == 0 || cs[0].rule != "date-swap" {
			t.Fatalf("candidates = %+v", cs)
		}
		if !cs[0].params.DeadlineFrom.Equal(*date("2026-08-10")) {
			t.Errorf("swap not applied: %v", cs[0].params.DeadlineFrom)
		}
	})

	t.Run("deadline entirely in the past", func(t *testing.T) {
		p := models.SearchParams{DeadlineFrom: date("2026-01-01"), DeadlineTo: date("2026-02-01")}
		cs := staticCandidates(p, parseQuery(""), "", today)
		found := false
		for _, c := range cs {
			if c.rule == "deadline-past" && c.params.DeadlineTo == nil {
				found = true
			}
		}
		if !found {
			t.Fatalf("no deadline-past candidate in %+v", cs)
		}
	})

	t.Run("solicitation number with filters", func(t *testing.T) {
		p := models.SearchParams{Query: "SPE4A626T13GL", NAICSCodes: []string{"541611"}}
		cs := staticCandidates(p, parseQuery(p.Query), "", today)
		found := false
		for _, c := range cs {
			if c.rule == "solnum-unfiltered" && len(c.params.NAICSCodes) == 0 && c.params.Query == p.Query {
				found = true
			}
		}
		if !found {
			t.Fatalf("no solnum candidate in %+v", cs)
		}
	})

	t.Run("quoted phrase produces unquote", func(t *testing.T) {
		p := models.SearchParams{Query: `"janitorial services"`}
		cs := staticCandidates(p, parseQuery(p.Query), "", today)
		if len(cs) != 1 || cs[0].rule != "unquote" || cs[0].params.Query != "janitorial services" {
			t.Fatalf("candidates = %+v", cs)
		}
	})

	t.Run("spelling fix", func(t *testing.T) {
		p := models.SearchParams{Query: "janitoral"}
		cs := staticCandidates(p, parseQuery(p.Query), "janitorial", today)
		if len(cs) != 1 || cs[0].rule != "typo" || cs[0].params.Query != "janitorial" {
			t.Fatalf("candidates = %+v", cs)
		}
	})
}

func TestRescueUnquote(t *testing.T) {
	prober := &fakeProber{countFn: func(p models.SearchParams) int {
		if p.Query == "janitorial services" {
			return 41
		}
		return 0
	}}
	s := newTestService(prober, nil)

	res := s.Rescue(context.Background(), models.SearchParams{Query: `"janitorial services"`}, 0, "")
	if res == nil || res.Stage != "probe" {
		t.Fatalf("result = %+v", res)
	}
	if res.Cause != "quoted-phrase" {
		t.Errorf("cause = %s", res.Cause)
	}
	if len(res.Suggestions) == 0 {
		t.Fatal("no suggestions")
	}
	sug := res.Suggestions[0]
	if sug.Rule != "unquote" || sug.VerifiedTotal != 41 || sug.Params["q"] != "janitorial services" {
		t.Errorf("suggestion = %+v", sug)
	}
}

func TestRescueOriginalActuallyHasResults(t *testing.T) {
	prober := &fakeProber{countFn: func(models.SearchParams) int { return 7 }}
	s := newTestService(prober, nil)
	if res := s.Rescue(context.Background(), models.SearchParams{Query: "anything"}, 0, ""); res != nil {
		t.Fatalf("expected nil for a search that has results, got %+v", res)
	}
}

func TestRescueFilterFatal(t *testing.T) {
	prober := &fakeProber{countFn: func(p models.SearchParams) int {
		switch {
		case p.Query == "janitorial" && len(p.NAICSCodes) == 0 && p.NAICSPrefix == "":
			return 87
		case p.Query == "janitorial" && p.NAICSPrefix == "5416":
			return 23
		default:
			return 0
		}
	}}
	s := newTestService(prober, nil)

	orig := models.SearchParams{Query: "janitorial", NAICSCodes: []string{"541611"}}
	res := s.Rescue(context.Background(), orig, 0, "")
	if res == nil || res.Stage != "probe" {
		t.Fatalf("result = %+v", res)
	}

	byRule := map[string]Suggestion{}
	for _, sug := range res.Suggestions {
		byRule[sug.Rule] = sug
	}
	family, ok := byRule["naics-family"]
	if !ok || family.VerifiedTotal != 23 || family.Params["naics_prefix"] != "5416" {
		t.Errorf("naics-family = %+v (present %v)", family, ok)
	}
	drop, ok := byRule["drop-naics"]
	if !ok || drop.VerifiedTotal != 87 {
		t.Errorf("drop-naics = %+v (present %v)", drop, ok)
	}
}

func TestRescueProbeBudget(t *testing.T) {
	prober := &fakeProber{countFn: func(models.SearchParams) int { return 0 }}
	s := newTestService(prober, nil)

	orig := models.SearchParams{
		Query:      "one two three four",
		NAICSCodes: []string{"541611"},
		SetAsides:  []string{"SBA"},
		States:     []string{"VA"},
		PSCCodes:   []string{"S201"},
	}
	res := s.Rescue(context.Background(), orig, 0, "")
	if res == nil || res.Stage != "none" || len(res.Suggestions) != 0 {
		t.Fatalf("result = %+v", res)
	}
	if prober.calls > maxProbes {
		t.Errorf("prober called %d times, budget is %d", prober.calls, maxProbes)
	}
}

func TestRescueCaches(t *testing.T) {
	prober := &fakeProber{countFn: func(p models.SearchParams) int {
		if p.Query == "janitorial services" {
			return 41
		}
		return 0
	}}
	s := newTestService(prober, nil)

	orig := models.SearchParams{Query: `"janitorial services"`}
	first := s.Rescue(context.Background(), orig, 0, "")
	callsAfterFirst := prober.calls
	second := s.Rescue(context.Background(), orig, 0, "")
	if prober.calls != callsAfterFirst {
		t.Errorf("second rescue probed again: %d -> %d calls", callsAfterFirst, prober.calls)
	}
	if first == nil || second == nil || first.Suggestions[0].VerifiedTotal != second.Suggestions[0].VerifiedTotal {
		t.Errorf("cached result differs: %+v vs %+v", first, second)
	}
}

func TestRescueNothingToRescue(t *testing.T) {
	prober := &fakeProber{countFn: func(models.SearchParams) int { return 0 }}
	s := newTestService(prober, nil)
	if res := s.Rescue(context.Background(), models.SearchParams{}, 0, ""); res != nil {
		t.Fatalf("expected nil for empty params, got %+v", res)
	}
	if prober.calls != 0 {
		t.Errorf("prober called %d times for empty params", prober.calls)
	}
}

// A probe must model what the user actually sees. The frontend's "still open"
// default is active=true, which INCLUDES notices with no stated deadline —
// probing with a bare deadline_from instead undercounts by every undated
// notice, and suggestions get dropped as false zeros.
func TestNormalizeForProbe_UsesActiveOnlyNotDeadlineFrom(t *testing.T) {
	s := &Service{}
	got := s.normalizeForProbe(models.SearchParams{Query: "coffee"})

	if !got.ActiveOnly {
		t.Error("bare params must probe with ActiveOnly set")
	}
	if got.DeadlineFrom != nil {
		t.Errorf("bare params must NOT get a deadline_from default, got %v", got.DeadlineFrom)
	}
	if m := paramsToMap(got); m["active"] != "true" || m["deadline_from"] != "" {
		t.Errorf("serialized probe params wrong: %v", m)
	}
}

// An explicit deadline range is a different intent and must be left alone.
func TestNormalizeForProbe_LeavesExplicitDeadlineRange(t *testing.T) {
	s := &Service{}
	got := s.normalizeForProbe(models.SearchParams{
		Query: "coffee", DeadlineFrom: date("2026-09-01"),
	})

	if got.ActiveOnly {
		t.Error("an explicit deadline range must not be turned into ActiveOnly")
	}
	if got.DeadlineFrom == nil || !got.DeadlineFrom.Equal(*date("2026-09-01")) {
		t.Errorf("explicit deadline_from was altered: %v", got.DeadlineFrom)
	}
}

// active must survive the map round-trip or probe dedupe collides.
func TestParamsMapRoundTrip_PreservesActiveOnly(t *testing.T) {
	in := models.SearchParams{Query: "coffee", ActiveOnly: true}
	out := paramsFromMap(paramsToMap(in))

	if !out.ActiveOnly {
		t.Error("ActiveOnly lost in map round-trip")
	}
	if canonical(in) == canonical(models.SearchParams{Query: "coffee"}) {
		t.Error("active=true and bare params must not share a canonical key")
	}
}

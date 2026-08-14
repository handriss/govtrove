// Package searchrescue diagnoses zero-result searches and proposes
// alternative searches verified to return results. Cost-tiered: static
// inspection and probed deterministic rewrites resolve the common causes
// without any LLM; the LLM tier handles only the long tail, and its
// suggestions pass the same server-side count verification as everything
// else, so an unverified count can never reach the user.
package searchrescue

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

const (
	maxProbes      = 8
	maxSuggestions = 3
	llmDailyCap    = 200
)

type Prober interface {
	Count(ctx context.Context, params models.SearchParams) (int, error)
}

// ExpandFunc mirrors the search handler's geo-synonym expansion so probe
// counts match what the app would actually show for the same params.
type ExpandFunc func(ctx context.Context, p *models.SearchParams)

type Suggestion struct {
	Label         string            `json:"label"`
	Explanation   string            `json:"explanation,omitempty"`
	Params        map[string]string `json:"params"`
	VerifiedTotal int               `json:"verified_total"`
	Rule          string            `json:"rule"`
}

type Result struct {
	Stage       string       `json:"stage"` // "probe", "llm", or "none"
	Cause       string       `json:"cause,omitempty"`
	Explanation string       `json:"explanation,omitempty"`
	Suggestions []Suggestion `json:"suggestions"`
}

type Service struct {
	prober Prober
	llm    *LLMClient // nil disables the LLM tier; deterministic tiers still run
	expand ExpandFunc
	logger *slog.Logger
	cache  *cache
	now    func() time.Time

	llmDay   string
	llmCalls int
}

func New(prober Prober, llm *LLMClient, expand ExpandFunc, logger *slog.Logger) *Service {
	return &Service{
		prober: prober,
		llm:    llm,
		expand: expand,
		logger: logger,
		cache:  newCache(6*time.Hour, 500),
		now:    time.Now,
	}
}

// significant is the single significance rule for every tier: for a
// zero-result original any hit counts; in low-result mode the suggestion
// must be a real improvement, not a marginal one.
func significant(orig, n int) bool {
	if orig == 0 {
		return n >= 1
	}
	return n >= 10 && n >= 3*orig
}

type budget struct{ remaining int }

type probeEntry struct {
	Params map[string]string `json:"params"`
	Total  int               `json:"total"`
}

type probeLog struct {
	entries []probeEntry
	seen    map[string]int
}

func newProbeLog() *probeLog {
	return &probeLog{seen: make(map[string]int)}
}

// Rescue runs the cascade. Returns nil when there is nothing to rescue
// (the original search actually has results, or no query/filters at all).
func (s *Service) Rescue(ctx context.Context, orig models.SearchParams, origTotal int, spellFix string) *Result {
	if orig.Query == "" && !hasRelaxableFilters(orig) {
		return nil
	}

	key := canonical(s.normalizeForProbe(orig))
	if cached, ok := s.cache.get(key); ok {
		return cached
	}

	b := &budget{remaining: maxProbes}
	pl := newProbeLog()

	// Re-verify the zero before diagnosing it: the client's state may be
	// stale, and this count doubles as the bisect's "together" measurement.
	together, ok := s.probe(ctx, b, pl, orig)
	if !ok {
		return nil
	}
	if significant(origTotal, together) {
		return nil
	}

	result := s.cascade(ctx, b, pl, orig, origTotal, spellFix)
	s.cache.set(key, result)
	return result
}

func (s *Service) cascade(ctx context.Context, b *budget, pl *probeLog, orig models.SearchParams, origTotal int, spellFix string) *Result {
	pq := parseQuery(orig.Query)
	var accepted []Suggestion

	verify := func(c candidate) {
		if len(accepted) >= maxSuggestions || b.remaining == 0 {
			return
		}
		n, ok := s.probe(ctx, b, pl, c.params)
		if !ok || !significant(origTotal, n) {
			return
		}
		accepted = append(accepted, Suggestion{
			Label:         c.label,
			Explanation:   fmt.Sprintf(c.explainFmt, n),
			Params:        paramsToMap(c.params),
			VerifiedTotal: n,
			Rule:          c.rule,
		})
	}

	for _, c := range staticCandidates(orig, pq, spellFix, s.today()) {
		verify(c)
	}

	if len(accepted) < maxSuggestions && b.remaining > 0 {
		queryFatal, filterFatal := s.bisect(ctx, b, pl, orig)
		for _, c := range relaxationCandidates(orig, pq, queryFatal, filterFatal) {
			verify(c)
		}
	}

	if len(accepted) > 0 {
		return &Result{
			Stage:       "probe",
			Cause:       causeForRule(accepted[0].Rule),
			Explanation: accepted[0].Explanation,
			Suggestions: accepted,
		}
	}

	if s.llm != nil && b.remaining > 0 && s.llmAllowed() {
		if res := s.llmRescue(ctx, b, pl, orig, origTotal, spellFix); res != nil {
			return res
		}
	}

	return &Result{Stage: "none", Cause: "no-market", Suggestions: []Suggestion{}}
}

// bisect splits blame between the query and the filters so the candidate
// budget is spent on the side that is actually fatal.
func (s *Service) bisect(ctx context.Context, b *budget, pl *probeLog, orig models.SearchParams) (queryFatal, filterFatal bool) {
	hasFilters := hasRelaxableFilters(orig)
	if orig.Query == "" {
		return false, true
	}
	if !hasFilters {
		return true, false
	}
	if b.remaining < 2 {
		return true, true
	}

	queryOnly := models.SearchParams{Query: orig.Query}
	if n, ok := s.probe(ctx, b, pl, queryOnly); ok && n == 0 {
		return true, false
	}
	filtersOnly := orig
	filtersOnly.Query = ""
	if n, ok := s.probe(ctx, b, pl, filtersOnly); ok && n == 0 {
		return false, true
	}
	// Both sides work alone; the combination is fatal — relax filters first
	// (dropping one keeps more of the user's intent than rewriting words).
	return false, true
}

func (s *Service) probe(ctx context.Context, b *budget, pl *probeLog, p models.SearchParams) (int, bool) {
	if b.remaining <= 0 {
		return 0, false
	}
	b.remaining--

	normalized := s.normalizeForProbe(p)
	key := canonical(normalized)
	if n, ok := pl.seen[key]; ok {
		b.remaining++ // cache hit, no query spent
		return n, true
	}

	if s.expand != nil && normalized.Query != "" {
		s.expand(ctx, &normalized)
	}

	n, err := s.prober.Count(ctx, normalized)
	if err != nil {
		s.logger.Warn("rescue probe failed", "error", err)
		return 0, false
	}
	pl.seen[key] = n
	pl.entries = append(pl.entries, probeEntry{Params: paramsToMap(p), Total: n})
	return n, true
}

// normalizeForProbe applies the frontend's implicit defaults so a probe
// counts what the user would actually see. A bare params set must probe the
// same way or counts misstate reality (the 810-quoted-vs-22-shown bug).
//
// The default is ActiveOnly, NOT deadline_from=today: "still open" includes
// notices with no deadline at all. Probing with a bare date would undercount
// by the ~8k undated active notices the real search returns.
func (s *Service) normalizeForProbe(p models.SearchParams) models.SearchParams {
	p.Sort, p.Order, p.Page, p.Limit = "", "", 0, 0
	p.GeoStates, p.GeoCities = nil, nil
	if p.DeadlineFrom == nil && p.DeadlineTo == nil {
		p.ActiveOnly = true
	}
	return p
}

func (s *Service) today() time.Time {
	now := s.now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) llmAllowed() bool {
	day := s.now().Format("2006-01-02")
	if s.llmDay != day {
		s.llmDay, s.llmCalls = day, 0
	}
	if s.llmCalls >= llmDailyCap {
		s.logger.Warn("rescue LLM daily cap reached", "cap", llmDailyCap)
		return false
	}
	s.llmCalls++
	return true
}

func hasRelaxableFilters(p models.SearchParams) bool {
	return len(p.Types) > 0 || len(p.SetAsides) > 0 ||
		len(p.NAICSCodes) > 0 || p.NAICSPrefix != "" || len(p.NAICSPrefixes) > 0 ||
		len(p.PSCCodes) > 0 || p.PSCPrefix != "" || len(p.PSCPrefixes) > 0 ||
		len(p.States) > 0 || p.Department != "" || len(p.AgencyPaths) > 0 ||
		p.PostedFrom != nil || p.PostedTo != nil ||
		p.DeadlineFrom != nil || p.DeadlineTo != nil ||
		p.PopCity != ""
}

const dateFmt = "2006-01-02"

// paramsToMap serializes params with the same names the URL and the search
// API use, so the frontend can apply a suggestion by plain navigation.
func paramsToMap(p models.SearchParams) map[string]string {
	m := make(map[string]string)
	set := func(k, v string) {
		if v != "" {
			m[k] = v
		}
	}
	set("q", p.Query)
	set("type", strings.Join(p.Types, ","))
	set("set_aside", strings.Join(p.SetAsides, ","))
	set("naics", strings.Join(p.NAICSCodes, ","))
	set("naics_prefix", p.NAICSPrefix)
	set("naics_prefixes", strings.Join(p.NAICSPrefixes, ","))
	set("psc", strings.Join(p.PSCCodes, ","))
	set("psc_prefix", p.PSCPrefix)
	set("psc_prefixes", strings.Join(p.PSCPrefixes, ","))
	set("state", strings.Join(p.States, ","))
	set("department", p.Department)
	set("agency", strings.Join(p.AgencyPaths, ","))
	set("sol_num", p.SolicitationNumber)
	set("pop_city", p.PopCity)
	if p.PostedFrom != nil {
		set("posted_from", p.PostedFrom.Format(dateFmt))
	}
	if p.PostedTo != nil {
		set("posted_to", p.PostedTo.Format(dateFmt))
	}
	if p.DeadlineFrom != nil {
		set("deadline_from", p.DeadlineFrom.Format(dateFmt))
	}
	if p.DeadlineTo != nil {
		set("deadline_to", p.DeadlineTo.Format(dateFmt))
	}
	if p.ActiveOnly {
		set("active", "true")
	}
	return m
}

func paramsFromMap(m map[string]string) models.SearchParams {
	var p models.SearchParams
	split := func(k string) []string {
		if m[k] == "" {
			return nil
		}
		return strings.Split(m[k], ",")
	}
	date := func(k string) *time.Time {
		if m[k] == "" {
			return nil
		}
		t, err := time.Parse(dateFmt, m[k])
		if err != nil {
			return nil
		}
		return &t
	}
	p.Query = m["q"]
	p.Types = split("type")
	p.SetAsides = split("set_aside")
	p.NAICSCodes = split("naics")
	p.NAICSPrefix = m["naics_prefix"]
	p.NAICSPrefixes = split("naics_prefixes")
	p.PSCCodes = split("psc")
	p.PSCPrefix = m["psc_prefix"]
	p.PSCPrefixes = split("psc_prefixes")
	p.States = split("state")
	p.Department = m["department"]
	p.AgencyPaths = split("agency")
	p.SolicitationNumber = m["sol_num"]
	p.PopCity = m["pop_city"]
	p.PostedFrom = date("posted_from")
	p.PostedTo = date("posted_to")
	p.DeadlineFrom = date("deadline_from")
	p.DeadlineTo = date("deadline_to")
	p.ActiveOnly = m["active"] == "true"
	return p
}

func canonical(p models.SearchParams) string {
	m := paramsToMap(p)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(m[k])
		sb.WriteByte('&')
	}
	return sb.String()
}

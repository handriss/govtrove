package searchrescue

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

type candidate struct {
	params     models.SearchParams
	rule       string
	cause      string
	label      string
	explainFmt string // one %d placeholder for the verified count
}

func causeForRule(rule string) string {
	switch rule {
	case "unquote":
		return "quoted-phrase"
	case "typo":
		return "typo"
	case "drop-term":
		return "too-many-terms"
	case "date-swap", "deadline-past", "posted-future", "drop-deadline":
		return "deadline-window"
	case "solnum-unfiltered":
		return "filters-exclude-record"
	case "naics-conflict", "naics-family", "drop-naics":
		return "naics-mismatch"
	case "drop-psc":
		return "psc-mismatch"
	case "drop-set-aside":
		return "set-aside-too-narrow"
	case "drop-type":
		return "type-mismatch"
	case "drop-state", "drop-pop-city":
		return "location-mismatch"
	case "drop-agency", "drop-department":
		return "agency-mismatch"
	}
	return "no-market"
}

var (
	solNumRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\-]{5,}$`)
	naicsLikeRe = regexp.MustCompile(`\b\d{6}\b`)
)

func looksLikeSolicitationNumber(q string) bool {
	return solNumRe.MatchString(q) &&
		strings.ContainsAny(q, "0123456789") &&
		(strings.ContainsAny(q, "-._") || strings.IndexFunc(q, func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }) >= 0)
}

// staticCandidates are the near-certain fixes detectable by inspection alone.
func staticCandidates(orig models.SearchParams, pq parsedQuery, spellFix string, today time.Time) []candidate {
	var out []candidate

	if orig.DeadlineFrom != nil && orig.DeadlineTo != nil && orig.DeadlineFrom.After(*orig.DeadlineTo) {
		p := orig
		p.DeadlineFrom, p.DeadlineTo = p.DeadlineTo, p.DeadlineFrom
		out = append(out, candidate{p, "date-swap", "deadline-window",
			"Swap the deadline dates",
			"Your deadline range was reversed — swapped around, it finds %d opportunities."})
	}
	if orig.PostedFrom != nil && orig.PostedTo != nil && orig.PostedFrom.After(*orig.PostedTo) {
		p := orig
		p.PostedFrom, p.PostedTo = p.PostedTo, p.PostedFrom
		out = append(out, candidate{p, "date-swap", "deadline-window",
			"Swap the posted dates",
			"Your posted-date range was reversed — swapped around, it finds %d opportunities."})
	}
	if orig.DeadlineTo != nil && orig.DeadlineTo.Before(today) {
		p := orig
		p.DeadlineFrom, p.DeadlineTo = nil, nil
		out = append(out, candidate{p, "deadline-past", "deadline-window",
			"Show upcoming deadlines",
			"Your deadline filter only allowed past dates, and only opportunities you can still bid on are shown — with upcoming deadlines there are %d."})
	}
	if orig.PostedFrom != nil && orig.PostedFrom.After(today) {
		p := orig
		p.PostedFrom, p.PostedTo = nil, nil
		out = append(out, candidate{p, "posted-future", "deadline-window",
			"Remove the posted-date filter",
			"Your posted-after date is in the future — without it there are %d opportunities."})
	}
	if orig.Query != "" && looksLikeSolicitationNumber(orig.Query) && hasRelaxableFilters(orig) {
		out = append(out, candidate{models.SearchParams{Query: orig.Query}, "solnum-unfiltered", "filters-exclude-record",
			"Search the number without filters",
			"That looks like a solicitation number — searching it on its own finds %d."})
	}
	if len(orig.NAICSCodes) > 0 {
		if code := naicsLikeRe.FindString(orig.Query); code != "" && !contains(orig.NAICSCodes, code) {
			p := orig
			p.NAICSCodes, p.NAICSPrefix, p.NAICSPrefixes = nil, "", nil
			out = append(out, candidate{p, "naics-conflict", "naics-mismatch",
				"Drop the conflicting NAICS filter",
				"Your search mentions a NAICS code that conflicts with the NAICS filter — without the filter: %d results."})
		}
	}
	if len(pq.phrases) > 0 {
		p := orig
		p.Query = stripQuotes(orig.Query)
		out = append(out, candidate{p, "unquote", "quoted-phrase",
			"Match all words instead of the exact phrase",
			"Quotes require that exact phrase to appear word-for-word — matching all the words instead finds %d."})
	}
	if spellFix != "" && !strings.EqualFold(spellFix, orig.Query) {
		p := orig
		p.Query = spellFix
		out = append(out, candidate{p, "typo", "typo",
			fmt.Sprintf("Search %q instead", spellFix),
			fmt.Sprintf("%q matches %%d opportunities.", spellFix)})
	}
	return out
}

// relaxationCandidates widen the fatal side found by the bisect, smallest
// change first.
func relaxationCandidates(orig models.SearchParams, pq parsedQuery, queryFatal, filterFatal bool) []candidate {
	var out []candidate

	if queryFatal && len(pq.phrases) == 0 && !pq.hasOR && len(pq.terms) >= 2 && len(pq.terms) <= 5 {
		for i := len(pq.terms) - 1; i >= 0; i-- {
			rest := make([]string, 0, len(pq.terms)-1)
			rest = append(rest, pq.terms[:i]...)
			rest = append(rest, pq.terms[i+1:]...)
			p := orig
			p.Query = strings.Join(rest, " ")
			out = append(out, candidate{p, "drop-term", "too-many-terms",
				fmt.Sprintf("Search without %q", pq.terms[i]),
				fmt.Sprintf("Every word must appear somewhere — without %q there are %%d results.", pq.terms[i])})
		}
	}

	if !filterFatal {
		return out
	}

	if orig.DeadlineTo != nil {
		p := orig
		p.DeadlineFrom, p.DeadlineTo = nil, nil
		out = append(out, candidate{p, "drop-deadline", "deadline-window",
			"Remove the deadline filter",
			"Removing the deadline window finds %d opportunities."})
	}
	if len(orig.NAICSCodes) == 1 && len(orig.NAICSCodes[0]) == 6 {
		p := orig
		p.NAICSPrefix = orig.NAICSCodes[0][:4]
		p.NAICSCodes = nil
		out = append(out, candidate{p, "naics-family", "naics-mismatch",
			fmt.Sprintf("Broaden to the NAICS %s family", orig.NAICSCodes[0][:4]),
			fmt.Sprintf("Nothing is open under the exact code right now — the broader %s family has %%d.", orig.NAICSCodes[0][:4])})
	}

	type filterDrop struct {
		rule, label, explain string
		present              bool
		apply                func(*models.SearchParams)
	}
	drops := []filterDrop{
		{"drop-naics", "Remove the NAICS filter", "Removing the NAICS filter finds %d.",
			len(orig.NAICSCodes) > 0 || orig.NAICSPrefix != "" || len(orig.NAICSPrefixes) > 0,
			func(p *models.SearchParams) { p.NAICSCodes, p.NAICSPrefix, p.NAICSPrefixes = nil, "", nil }},
		{"drop-psc", "Remove the PSC filter", "Removing the PSC filter finds %d.",
			len(orig.PSCCodes) > 0 || orig.PSCPrefix != "" || len(orig.PSCPrefixes) > 0,
			func(p *models.SearchParams) { p.PSCCodes, p.PSCPrefix, p.PSCPrefixes = nil, "", nil }},
		{"drop-set-aside", "Remove the set-aside filter", "Removing the set-aside filter finds %d.",
			len(orig.SetAsides) > 0,
			func(p *models.SearchParams) { p.SetAsides = nil }},
		// No drop-type candidate: the app always sends its default notice
		// types, and omitting `type` in a suggestion just restores those
		// defaults on click — the probe's count could never match the UI.
		{"drop-state", "Remove the state filter", "Removing the state filter finds %d.",
			len(orig.States) > 0,
			func(p *models.SearchParams) { p.States = nil }},
		{"drop-agency", "Remove the agency filter", "Removing the agency filter finds %d.",
			len(orig.AgencyPaths) > 0,
			func(p *models.SearchParams) { p.AgencyPaths = nil }},
		{"drop-department", "Remove the department filter", "Removing the department filter finds %d.",
			orig.Department != "",
			func(p *models.SearchParams) { p.Department = "" }},
		{"drop-pop-city", "Remove the city filter", "Removing the place-of-performance city finds %d.",
			orig.PopCity != "",
			func(p *models.SearchParams) { p.PopCity = "" }},
	}
	for _, d := range drops {
		if !d.present {
			continue
		}
		p := orig
		d.apply(&p)
		out = append(out, candidate{p, d.rule, causeForRule(d.rule), d.label, d.explain})
	}
	return out
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

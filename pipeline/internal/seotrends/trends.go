package seotrends

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/naics"
)

type TrendsOutput struct {
	GeneratedAt    time.Time       `json:"generated_at"`
	NAICSMovers    []NAICSMover    `json:"naics_movers"`
	AgencyMovers   []AgencyMover   `json:"agency_movers"`
	TitlePhrases   []TitlePhrase   `json:"title_phrases"`
	SetAsideCounts []SetAsideCount `json:"set_aside_counts"`
}

type NAICSMover struct {
	Code     string  `json:"code"`
	Label    string  `json:"label"`
	ThisWeek int     `json:"this_week"`
	LastWeek int     `json:"last_week"`
	DeltaPct float64 `json:"delta_pct"`
	Score    float64 `json:"score"`
}

type AgencyMover struct {
	Department string  `json:"department"`
	ThisWeek   int     `json:"this_week"`
	LastWeek   int     `json:"last_week"`
	DeltaPct   float64 `json:"delta_pct"`
	Score      float64 `json:"score"`
}

type TitlePhrase struct {
	Phrase   string  `json:"phrase"`
	ThisWeek int     `json:"this_week"`
	LastWeek int     `json:"last_week"`
	DeltaPct float64 `json:"delta_pct"`
}

type SetAsideCount struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

func Analyze(
	naicsVolumes []database.NAICSVolume,
	agencyVolumes []database.AgencyVolume,
	titles []database.TitleRow,
	setAsides []database.SetAsideCount,
) *TrendsOutput {
	return &TrendsOutput{
		GeneratedAt:    time.Now().UTC(),
		NAICSMovers:    rankNAICS(naicsVolumes),
		AgencyMovers:   rankAgencies(agencyVolumes),
		TitlePhrases:   extractTitlePhrases(titles),
		SetAsideCounts: convertSetAsides(setAsides),
	}
}

func deltaPct(thisWeek, lastWeek int) float64 {
	if lastWeek == 0 {
		if thisWeek == 0 {
			return 0
		}
		return 999.0
	}
	return float64(thisWeek-lastWeek) / float64(lastWeek) * 100
}

func compositeScore(thisWeek int, pct float64) float64 {
	if thisWeek <= 0 {
		return 0
	}
	return pct * math.Log(float64(thisWeek))
}

func rankNAICS(volumes []database.NAICSVolume) []NAICSMover {
	movers := make([]NAICSMover, 0, len(volumes))
	for _, v := range volumes {
		pct := deltaPct(v.ThisWeek, v.LastWeek)
		movers = append(movers, NAICSMover{
			Code:     v.NAICSCode,
			Label:    naics.Label(v.NAICSCode),
			ThisWeek: v.ThisWeek,
			LastWeek: v.LastWeek,
			DeltaPct: math.Round(pct*10) / 10,
			Score:    math.Round(compositeScore(v.ThisWeek, pct)*10) / 10,
		})
	}
	sort.Slice(movers, func(i, j int) bool {
		return movers[i].Score > movers[j].Score
	})
	if len(movers) > 50 {
		movers = movers[:50]
	}
	return movers
}

func rankAgencies(volumes []database.AgencyVolume) []AgencyMover {
	movers := make([]AgencyMover, 0, len(volumes))
	for _, v := range volumes {
		pct := deltaPct(v.ThisWeek, v.LastWeek)
		movers = append(movers, AgencyMover{
			Department: v.Department,
			ThisWeek:   v.ThisWeek,
			LastWeek:   v.LastWeek,
			DeltaPct:   math.Round(pct*10) / 10,
			Score:      math.Round(compositeScore(v.ThisWeek, pct)*10) / 10,
		})
	}
	sort.Slice(movers, func(i, j int) bool {
		return movers[i].Score > movers[j].Score
	})
	if len(movers) > 50 {
		movers = movers[:50]
	}
	return movers
}

func convertSetAsides(sas []database.SetAsideCount) []SetAsideCount {
	out := make([]SetAsideCount, len(sas))
	for i, sa := range sas {
		out[i] = SetAsideCount{
			Code:        sa.SetAsideCode,
			Description: sa.SetAsideDescription,
			Count:       sa.Count,
		}
	}
	return out
}

// N-gram extraction

var procurementPrefixes = []string{
	"sources sought",
	"request for information",
	"request for proposal",
	"request for quote",
	"request for quotation",
	"synopsis",
	"amendment",
	"modification",
	"justification and approval",
	"combined synopsis/solicitation",
	"combined synopsis solicitation",
	"j&a",
	"rfi",
	"rfp",
	"rfq",
	"presolicitation",
	"special notice",
	"intent to sole source",
	"sole source",
	"notice of intent",
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "shall": true, "can": true, "this": true, "that": true,
	"these": true, "those": true, "it": true, "its": true, "as": true,
	"not": true, "no": true, "all": true, "any": true, "each": true,
	"new": true, "per": true, "via": true,
	// Procurement jargon stop words
	"services": true, "support": true, "contract": true, "provide": true,
	"various": true, "multiple": true, "required": true, "solicitation": true,
	"notice": true, "amendment": true,
}

func stripPrefixes(title string) string {
	lower := strings.ToLower(title)
	for _, prefix := range procurementPrefixes {
		if strings.HasPrefix(lower, prefix) {
			rest := title[len(prefix):]
			rest = strings.TrimLeft(rest, " -–—:/")
			if rest != "" {
				return rest
			}
		}
	}
	return title
}

func tokenize(title string) []string {
	title = stripPrefixes(title)
	lower := strings.ToLower(title)

	var tokens []string
	var current strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			word := current.String()
			current.Reset()
			if !stopWords[word] && len(word) > 1 {
				tokens = append(tokens, word)
			}
		}
	}
	if current.Len() > 0 {
		word := current.String()
		if !stopWords[word] && len(word) > 1 {
			tokens = append(tokens, word)
		}
	}
	return tokens
}

func extractNgrams(tokens []string) []string {
	var ngrams []string
	for i := 0; i < len(tokens)-1; i++ {
		ngrams = append(ngrams, tokens[i]+" "+tokens[i+1])
	}
	for i := 0; i < len(tokens)-2; i++ {
		ngrams = append(ngrams, tokens[i]+" "+tokens[i+1]+" "+tokens[i+2])
	}
	return ngrams
}

func extractTitlePhrases(titles []database.TitleRow) []TitlePhrase {
	midpoint := time.Now().UTC().AddDate(0, 0, -7)

	thisWeekCounts := make(map[string]int)
	lastWeekCounts := make(map[string]int)

	// We don't have posted_date from the title rows, so we split the slice in half
	// as a proxy (titles are from 14-day window, roughly sorted by recency).
	// A more precise approach would require the posted_date, but for trend detection
	// this approximation is sufficient.
	_ = midpoint
	half := len(titles) / 2

	for i, t := range titles {
		tokens := tokenize(t.Title)
		ngrams := extractNgrams(tokens)
		seen := make(map[string]bool)
		for _, ng := range ngrams {
			if seen[ng] {
				continue
			}
			seen[ng] = true
			if i < half {
				thisWeekCounts[ng]++
			} else {
				lastWeekCounts[ng]++
			}
		}
	}

	var phrases []TitlePhrase
	for ng, tw := range thisWeekCounts {
		lw := lastWeekCounts[ng]
		if tw < 3 {
			continue
		}
		pct := deltaPct(tw, lw)
		phrases = append(phrases, TitlePhrase{
			Phrase:   ng,
			ThisWeek: tw,
			LastWeek: lw,
			DeltaPct: math.Round(pct*10) / 10,
		})
	}

	sort.Slice(phrases, func(i, j int) bool {
		return phrases[i].DeltaPct > phrases[j].DeltaPct
	})
	if len(phrases) > 30 {
		phrases = phrases[:30]
	}
	return phrases
}

package seotrends

import (
	"testing"

	"github.com/handriss/govtrove/pipeline/internal/database"
)

func TestDeltaPct(t *testing.T) {
	tests := []struct {
		name     string
		thisWeek int
		lastWeek int
		want     float64
	}{
		{"equal", 10, 10, 0},
		{"doubled", 20, 10, 100},
		{"halved", 5, 10, -50},
		{"new entry", 10, 0, 999},
		{"both zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deltaPct(tt.thisWeek, tt.lastWeek)
			if got != tt.want {
				t.Errorf("deltaPct(%d, %d) = %v, want %v", tt.thisWeek, tt.lastWeek, got, tt.want)
			}
		})
	}
}

func TestCompositeScore(t *testing.T) {
	score := compositeScore(100, 50)
	if score <= 0 {
		t.Errorf("expected positive score, got %v", score)
	}
	zero := compositeScore(0, 100)
	if zero != 0 {
		t.Errorf("expected zero score for zero volume, got %v", zero)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		title string
		want  []string
	}{
		{
			"Sources Sought - Cybersecurity Assessment Tools",
			[]string{"cybersecurity", "assessment", "tools"},
		},
		{
			"Combined Synopsis/Solicitation: IT Network Maintenance",
			[]string{"network", "maintenance"},
		},
		{
			"J&A - Sole Source Logistics",
			[]string{"sole", "source", "logistics"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := tokenize(tt.title)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %v, want %v", tt.title, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.title, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractNgrams(t *testing.T) {
	tokens := []string{"cybersecurity", "assessment", "tools"}
	ngrams := extractNgrams(tokens)
	expected := []string{
		"cybersecurity assessment",
		"assessment tools",
		"cybersecurity assessment tools",
	}
	if len(ngrams) != len(expected) {
		t.Fatalf("extractNgrams = %v, want %v", ngrams, expected)
	}
	for i := range ngrams {
		if ngrams[i] != expected[i] {
			t.Errorf("ngrams[%d] = %q, want %q", i, ngrams[i], expected[i])
		}
	}
}

func TestRankNAICS(t *testing.T) {
	volumes := []database.NAICSVolume{
		{NAICSCode: "541511", ThisWeek: 100, LastWeek: 50},
		{NAICSCode: "541512", ThisWeek: 10, LastWeek: 10},
		{NAICSCode: "541330", ThisWeek: 200, LastWeek: 20},
	}
	movers := rankNAICS(volumes)
	if len(movers) != 3 {
		t.Fatalf("expected 3 movers, got %d", len(movers))
	}
	// 541330 should rank first (highest delta + volume)
	if movers[0].Code != "541330" {
		t.Errorf("expected 541330 first, got %s", movers[0].Code)
	}
	// 541512 should be last (0% delta)
	if movers[2].Code != "541512" {
		t.Errorf("expected 541512 last, got %s", movers[2].Code)
	}
}

func TestRankAgencies(t *testing.T) {
	volumes := []database.AgencyVolume{
		{Department: "DEPT OF DEFENSE", ThisWeek: 500, LastWeek: 400},
		{Department: "DEPT OF ENERGY", ThisWeek: 50, LastWeek: 10},
	}
	movers := rankAgencies(volumes)
	if len(movers) != 2 {
		t.Fatalf("expected 2 movers, got %d", len(movers))
	}
}

func TestAnalyze(t *testing.T) {
	out := Analyze(nil, nil, nil, nil)
	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if out.GeneratedAt.IsZero() {
		t.Error("expected non-zero GeneratedAt")
	}
	if out.NAICSMovers == nil {
		t.Error("expected non-nil NAICSMovers slice")
	}
}

func TestExtractTitlePhrases(t *testing.T) {
	// Create enough titles with repeating phrases to pass the threshold
	titles := make([]database.TitleRow, 0, 20)
	for i := 0; i < 10; i++ {
		titles = append(titles, database.TitleRow{Title: "Cybersecurity Assessment Tools Procurement"})
	}
	for i := 0; i < 10; i++ {
		titles = append(titles, database.TitleRow{Title: "Office Supply Delivery"})
	}
	phrases := extractTitlePhrases(titles)
	if len(phrases) == 0 {
		t.Error("expected some phrases extracted")
	}
}

func TestStripPrefixes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Sources Sought - Test", "Test"},
		{"Amendment: Updated Deadline", "Updated Deadline"},
		{"Regular Title", "Regular Title"},
		{"REQUEST FOR INFORMATION: Analysis", "Analysis"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripPrefixes(tt.input)
			if got != tt.want {
				t.Errorf("stripPrefixes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

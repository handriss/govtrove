package ogimage

import (
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(14 * 24 * time.Hour)
	data := OpportunityData{
		ID:               12345,
		Title:            "Cybersecurity Assessment and Authorization Support Services for DoD Networks",
		Type:             "Solicitation",
		Department:       "DEPT OF DEFENSE.DEPT OF THE AIR FORCE.FA3016 – TINKER AFB",
		SetAsideDesc:     "Service-Disabled Veteran-Owned Small Business (SDVOSB) Set-Aside (FAR 19.14)",
		SetAsideCode:     "SDVOSBC",
		NAICSCode:        "541512",
		ResponseDeadline: &deadline,
	}

	png, err := r.Render(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(png) == 0 {
		t.Fatal("empty PNG output")
	}
	if len(png) > 200*1024 {
		t.Errorf("PNG too large: %d bytes", len(png))
	}

	// Verify PNG header
	if png[0] != 0x89 || png[1] != 'P' || png[2] != 'N' || png[3] != 'G' {
		t.Error("invalid PNG header")
	}

	t.Logf("Generated PNG: %d bytes", len(png))
}

func TestShortenAgency(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"DEPT OF DEFENSE.DEPT OF THE AIR FORCE.FA3016", "DoD — Air Force"},
		{"DEPT OF DEFENSE.DEPT OF THE ARMY.W56HZV", "DoD — Army"},
		{"DEPT OF DEFENSE.DEPT OF THE NAVY.N00024", "DoD — Navy"},
		{"GENERAL SERVICES ADMINISTRATION.FEDERAL ACQUISITION SERVICE", "GSA"},
		{"HEALTH AND HUMAN SERVICES, DEPARTMENT OF.CENTERS FOR DISEASE CONTROL", "HHS"},
		{"HOMELAND SECURITY, DEPARTMENT OF.CUSTOMS AND BORDER PROTECTION", "DHS"},
		{"VETERANS AFFAIRS, DEPARTMENT OF.VA MEDICAL CENTER", "VA"},
		{"DEPT OF DEFENSE.DEFENSE LOGISTICS AGENCY.DLA", "DoD — DLA"},
		{"JUSTICE, DEPARTMENT OF.FEDERAL PRISON SYSTEM", "DOJ"},
		{"INTERIOR, DEPARTMENT OF THE.BUREAU OF RECLAMATION", "DOI"},
		{"ENVIRONMENTAL PROTECTION AGENCY", "EPA"},
		{"NATIONAL SCIENCE FOUNDATION", "NSF"},
		{"", ""},
	}
	for _, tc := range tests {
		got := ShortenAgency(tc.input)
		if got != tc.want {
			t.Errorf("ShortenAgency(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestRenderClosingSoon(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * 24 * time.Hour)
	data := OpportunityData{
		ID:               99999,
		Title:            "Short deadline test",
		Type:             "Presolicitation",
		Department:       "GENERAL SERVICES ADMINISTRATION",
		ResponseDeadline: &deadline,
	}

	png, err := r.Render(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("empty PNG output")
	}
	t.Logf("Closing soon PNG: %d bytes", len(png))
}

func TestRenderMinimalData(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}

	data := OpportunityData{
		ID:    1,
		Title: "Test",
	}

	png, err := r.Render(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("empty PNG output")
	}
	t.Logf("Minimal PNG: %d bytes", len(png))
}

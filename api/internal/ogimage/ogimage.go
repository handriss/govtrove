package ogimage

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"image/png"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/Inter-Bold.ttf
var interBoldTTF []byte

//go:embed fonts/Inter-Regular.ttf
var interRegularTTF []byte

const (
	imgWidth  = 1200
	imgHeight = 630

	maxCacheEntries = 500
	cacheTTL        = 24 * time.Hour
)

type OpportunityData struct {
	ID               int
	Title            string
	Type             string
	Department       string
	SetAsideDesc     string
	SetAsideCode     string
	NAICSCode        string
	ResponseDeadline *time.Time
}

type Renderer struct {
	boldFont    *opentype.Font
	regularFont *opentype.Font

	mu    sync.Mutex
	cache map[int]cacheEntry
	sf    singleflight
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type singleflight struct {
	mu sync.Mutex
	m  map[int]*call
}

type call struct {
	wg  sync.WaitGroup
	val []byte
	err error
}

func (sf *singleflight) Do(key int, fn func() ([]byte, error)) ([]byte, error) {
	sf.mu.Lock()
	if sf.m == nil {
		sf.m = make(map[int]*call)
	}
	if c, ok := sf.m[key]; ok {
		sf.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &call{}
	c.wg.Add(1)
	sf.m[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()

	return c.val, c.err
}

func NewRenderer() (*Renderer, error) {
	bold, err := opentype.Parse(interBoldTTF)
	if err != nil {
		return nil, fmt.Errorf("parse bold font: %w", err)
	}
	regular, err := opentype.Parse(interRegularTTF)
	if err != nil {
		return nil, fmt.Errorf("parse regular font: %w", err)
	}
	return &Renderer{
		boldFont:    bold,
		regularFont: regular,
		cache:       make(map[int]cacheEntry),
	}, nil
}

func (r *Renderer) newFace(f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Sprintf("create font face: %v", err))
	}
	return face
}

func (r *Renderer) Render(opp OpportunityData) ([]byte, error) {
	r.mu.Lock()
	if entry, ok := r.cache[opp.ID]; ok && time.Now().Before(entry.expiresAt) {
		r.mu.Unlock()
		return entry.data, nil
	}
	r.mu.Unlock()

	data, err := r.sf.Do(opp.ID, func() ([]byte, error) {
		return r.render(opp)
	})
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if len(r.cache) >= maxCacheEntries {
		var oldestKey int
		var oldestTime time.Time
		first := true
		for k, v := range r.cache {
			if time.Now().After(v.expiresAt) {
				delete(r.cache, k)
				continue
			}
			if first || v.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiresAt
				first = false
			}
		}
		if len(r.cache) >= maxCacheEntries {
			delete(r.cache, oldestKey)
		}
	}
	r.cache[opp.ID] = cacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
	r.mu.Unlock()

	return data, nil
}

// --- Color palette ---

var (
	bgTop       = hexColor("#0F172A")
	bgBottom    = hexColor("#162033")
	sepColor    = hexColor("#2A3A4F")
	titleColor  = hexColor("#F1F5F9")
	agencyColor = hexColor("#94A3B8")
	subtleColor = hexColor("#64748B")
	wordmark    = hexColor("#E2E8F0")
	amberColor  = hexColor("#F59E0B")
	redColor    = hexColor("#EF4444")
	whiteColor  = color.RGBA{255, 255, 255, 255}
)

func (r *Renderer) render(opp OpportunityData) ([]byte, error) {
	dc := gg.NewContext(imgWidth, imgHeight)

	// 1. Gradient background
	for y := 0; y < imgHeight; y++ {
		t := float64(y) / float64(imgHeight)
		c := lerpColor(bgTop, bgBottom, t)
		dc.SetColor(c)
		dc.DrawRectangle(0, float64(y), imgWidth, 1)
		dc.Fill()
	}

	// 2. Left accent bar (4px, colored by type)
	dc.SetColor(typeAccentColor(opp.Type))
	dc.DrawRectangle(0, 0, 4, imgHeight)
	dc.Fill()

	// 3. Header zone (y=0..90)
	dc.SetFontFace(r.newFace(r.boldFont, 32))
	dc.SetColor(wordmark)
	drawSpacedString(dc, "GovTrove", 60, 55, 1.5)

	// Type badge (top right)
	if opp.Type != "" {
		typeBadge := strings.ToUpper(opp.Type)
		badgeColor := typeAccentColor(opp.Type)

		dc.SetFontFace(r.newFace(r.boldFont, 22))
		tw, _ := dc.MeasureString(typeBadge)
		padH := 20.0
		padV := 10.0
		badgeW := tw + padH*2
		badgeH := 22 + padV*2
		badgeX := float64(imgWidth) - 60 - badgeW
		badgeY := 45 - badgeH/2

		dc.SetColor(badgeColor)
		dc.DrawRoundedRectangle(badgeX, badgeY, badgeW, badgeH, 6)
		dc.Fill()

		dc.SetColor(whiteColor)
		dc.DrawString(typeBadge, badgeX+padH, badgeY+padV+18)
	}

	// Header separator
	dc.SetColor(sepColor)
	dc.DrawRectangle(0, 90, imgWidth, 1)
	dc.Fill()

	// 4. Title — dynamic font size, largest that fits in 3 lines
	title := normalizeTitle(opp.Title)
	dc.SetColor(titleColor)
	titleMaxW := 1080.0

	type titleSize struct {
		fontSize   float64
		lineHeight float64
	}
	sizes := []titleSize{
		{56, 72},
		{48, 64},
		{42, 56},
		{36, 48},
	}

	var lines []string
	var chosen titleSize
	for _, s := range sizes {
		dc.SetFontFace(r.newFace(r.boldFont, s.fontSize))
		lines = dc.WordWrap(title, titleMaxW)
		chosen = s
		if len(lines) <= 3 {
			break
		}
	}
	if len(lines) > 3 {
		lines[2] = lines[2] + "..."
		lines = lines[:3]
	}

	dc.SetFontFace(r.newFace(r.boldFont, chosen.fontSize))
	titleStartY := 190.0
	for _, line := range lines {
		dc.DrawString(line, 60, titleStartY)
		titleStartY += chosen.lineHeight
	}

	// 5. Agency line — below last title line
	agency := ShortenAgency(opp.Department)
	if agency != "" {
		dc.SetFontFace(r.newFace(r.regularFont, 32))
		dc.SetColor(agencyColor)
		dc.DrawString(agency, 60, titleStartY+28)
	}

	// 6. Footer zone (y=510..630)
	dc.SetColor(sepColor)
	dc.DrawRectangle(0, 510, imgWidth, 1)
	dc.Fill()

	footerY := 578.0
	xPos := 60.0

	// Set-aside badge
	setAsideLabel := shortenSetAside(opp.SetAsideCode, opp.SetAsideDesc)
	if setAsideLabel != "" {
		badgeColor := setAsideBadgeColor(opp.SetAsideCode)
		dc.SetFontFace(r.newFace(r.boldFont, 22))
		sw, _ := dc.MeasureString(setAsideLabel)
		padH := 16.0
		padV := 10.0
		badgeW := sw + padH*2
		badgeH := 22 + padV*2
		badgeY := footerY - badgeH/2 - 4

		dc.SetColor(badgeColor)
		dc.DrawRoundedRectangle(xPos, badgeY, badgeW, badgeH, 6)
		dc.Fill()

		dc.SetColor(whiteColor)
		dc.DrawString(setAsideLabel, xPos+padH, badgeY+padV+18)
		xPos += badgeW + 36
	}

	// Deadline
	if opp.ResponseDeadline != nil {
		dc.SetFontFace(r.newFace(r.regularFont, 28))
		deadlineStr, deadlineColor := formatDeadline(*opp.ResponseDeadline)
		dc.SetColor(deadlineColor)
		dc.DrawString(deadlineStr, xPos, footerY)
		dw, _ := dc.MeasureString(deadlineStr)
		xPos += dw + 44
	}

	// NAICS
	if opp.NAICSCode != "" {
		dc.SetFontFace(r.newFace(r.regularFont, 28))
		dc.SetColor(subtleColor)
		naicsStr := "NAICS: " + opp.NAICSCode
		dc.DrawString(naicsStr, xPos, footerY)
	}

	// govtrove.com — right-aligned
	dc.SetFontFace(r.newFace(r.regularFont, 24))
	dc.SetColor(hexColor("#475569"))
	urlStr := "govtrove.com"
	uw, _ := dc.MeasureString(urlStr)
	dc.DrawString(urlStr, float64(imgWidth)-60-uw, footerY)

	// Encode PNG
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// drawSpacedString renders a string with extra letter-spacing.
func drawSpacedString(dc *gg.Context, s string, x, y, spacing float64) {
	for _, ch := range s {
		str := string(ch)
		dc.DrawString(str, x, y)
		w, _ := dc.MeasureString(str)
		x += w + spacing
	}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: 255,
	}
}

// --- Title case normalization ---

var acronyms = map[string]string{
	"IT": "IT", "DOD": "DoD", "HVAC": "HVAC", "GPS": "GPS", "UAV": "UAV",
	"C4ISR": "C4ISR", "IDIQ": "IDIQ", "BPA": "BPA", "RFP": "RFP", "RFI": "RFI",
	"SOW": "SOW", "PWS": "PWS", "OCONUS": "OCONUS", "CONUS": "CONUS", "NATO": "NATO",
	"DISA": "DISA", "DLA": "DLA", "DARPA": "DARPA", "NASA": "NASA", "GSA": "GSA",
	"USACE": "USACE", "SPAWAR": "SPAWAR", "NAVAIR": "NAVAIR", "NAVSEA": "NAVSEA",
	"AFRL": "AFRL", "SOCOM": "SOCOM", "CENTCOM": "CENTCOM", "EUCOM": "EUCOM",
	"AFRICOM": "AFRICOM", "INDOPACOM": "INDOPACOM", "NORTHCOM": "NORTHCOM", "SOUTHCOM": "SOUTHCOM",
	"CYBERCOM": "CYBERCOM", "STRATCOM": "STRATCOM", "TRANSCOM": "TRANSCOM", "SPACECOM": "SPACECOM",
	"FCI": "FCI", "VRE": "VRE", "THAAD": "THAAD", "MDA": "MDA", "NGB": "NGB",
	"II": "II", "III": "III", "IV": "IV", "VI": "VI", "VII": "VII",
	"VIII": "VIII", "IX": "IX", "XI": "XI", "XII": "XII",
	"US": "US", "USA": "USA", "USAF": "USAF", "USN": "USN", "USMC": "USMC",
	"VA": "VA", "SBA": "SBA", "EPA": "EPA", "FBI": "FBI", "CIA": "CIA",
	"NSA": "NSA", "DHS": "DHS", "DOE": "DOE", "DOJ": "DOJ", "DOT": "DOT",
	"HHS": "HHS", "HUD": "HUD", "OPM": "OPM", "NRC": "NRC", "NSF": "NSF",
	"FEMA": "FEMA", "TSA": "TSA", "CBP": "CBP", "ICE": "ICE",
	"AFB": "AFB", "NAS": "NAS", "JBSA": "JBSA", "JBLM": "JBLM",
	"AC": "AC", "DC": "DC", "RF": "RF", "IP": "IP", "AI": "AI", "ML": "ML",
}

func normalizeTitle(title string) string {
	if !isAllCaps(title) {
		return title
	}
	words := strings.Fields(title)
	for i, w := range words {
		clean := strings.TrimRight(w, ".,;:!?-–—/()")
		suffix := w[len(clean):]
		prefix := ""
		if len(clean) > 0 && strings.ContainsAny(string(clean[0]), "(/") {
			prefix = string(clean[0])
			clean = clean[1:]
		}
		upper := strings.ToUpper(clean)
		if display, ok := acronyms[upper]; ok {
			words[i] = prefix + display + suffix
		} else if i == 0 || !isMinorWord(clean) {
			words[i] = prefix + toTitleWord(clean) + suffix
		} else {
			words[i] = prefix + strings.ToLower(clean) + suffix
		}
	}
	return strings.Join(words, " ")
}

func isAllCaps(s string) bool {
	upper, total := 0, 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			total++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	return total > 0 && float64(upper)/float64(total) > 0.6
}

func toTitleWord(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

var minorWords = map[string]bool{
	"A": true, "AN": true, "AND": true, "AS": true, "AT": true,
	"BUT": true, "BY": true, "FOR": true, "IN": true, "NOR": true,
	"OF": true, "ON": true, "OR": true, "SO": true, "THE": true,
	"TO": true, "UP": true, "YET": true, "WITH": true, "FROM": true,
}

func isMinorWord(w string) bool {
	return minorWords[strings.ToUpper(w)]
}

// --- Agency mapping ---

// ShortenAgency maps the dot-separated SAM.gov Department field to a readable short name.
func ShortenAgency(dept string) string {
	if dept == "" {
		return ""
	}

	parts := strings.SplitN(dept, ".", 3)
	topLevel := strings.TrimSpace(parts[0])

	agencyMap := map[string]string{
		"DEPT OF DEFENSE":                                "DoD",
		"GENERAL SERVICES ADMINISTRATION":                "GSA",
		"HEALTH AND HUMAN SERVICES, DEPARTMENT OF":       "HHS",
		"HOMELAND SECURITY, DEPARTMENT OF":               "DHS",
		"VETERANS AFFAIRS, DEPARTMENT OF":                 "VA",
		"TREASURY, DEPARTMENT OF THE":                    "Treasury",
		"JUSTICE, DEPARTMENT OF":                         "DOJ",
		"STATE, DEPARTMENT OF":                           "State Dept",
		"INTERIOR, DEPARTMENT OF THE":                    "DOI",
		"AGRICULTURE, DEPARTMENT OF":                     "USDA",
		"COMMERCE, DEPARTMENT OF":                        "Commerce",
		"EDUCATION, DEPARTMENT OF":                       "Education",
		"ENERGY, DEPARTMENT OF":                          "DOE",
		"LABOR, DEPARTMENT OF":                           "DOL",
		"TRANSPORTATION, DEPARTMENT OF":                  "DOT",
		"ENVIRONMENTAL PROTECTION AGENCY":                "EPA",
		"NATIONAL AERONAUTICS AND SPACE ADMINISTRATION":  "NASA",
		"SMALL BUSINESS ADMINISTRATION":                  "SBA",
		"SOCIAL SECURITY ADMINISTRATION":                 "SSA",
		"AGENCY FOR INTERNATIONAL DEVELOPMENT":           "USAID",
		"NUCLEAR REGULATORY COMMISSION":                  "NRC",
		"OFFICE OF PERSONNEL MANAGEMENT":                 "OPM",
		"NATIONAL SCIENCE FOUNDATION":                    "NSF",
		"EXECUTIVE OFFICE OF THE PRESIDENT":              "EOP",
	}

	subAgencyMap := map[string]string{
		"DEPT OF THE ARMY":                                "Army",
		"DEPT OF THE NAVY":                                "Navy",
		"DEPT OF THE AIR FORCE":                           "Air Force",
		"DEFENSE LOGISTICS AGENCY":                        "DLA",
		"DEFENSE INFORMATION SYSTEMS AGENCY":              "DISA",
		"DEFENSE HEALTH AGENCY":                           "DHA",
		"DEFENSE COUNTERINTELLIGENCE AND SECURITY AGENCY": "DCSA",
		"MISSILE DEFENSE AGENCY":                          "MDA",
		"DEFENSE ADVANCED RESEARCH PROJECTS AGENCY":       "DARPA",
		"NATIONAL GUARD BUREAU":                           "NGB",
		"US ARMY CORPS OF ENGINEERS":                      "USACE",
	}

	short := topLevel
	if mapped, ok := agencyMap[topLevel]; ok {
		short = mapped
	} else if len(topLevel) > 30 {
		short = topLevel[:30]
	}

	if len(parts) >= 2 {
		sub := strings.TrimSpace(parts[1])
		if mapped, ok := subAgencyMap[sub]; ok {
			if short == "DoD" {
				return short + " — " + mapped
			}
			return mapped
		}
	}

	return short
}

// --- Color helpers ---

func typeAccentColor(t string) color.Color {
	switch strings.ToLower(t) {
	case "solicitation":
		return hexColor("#2563EB")
	case "presolicitation":
		return hexColor("#0D9488")
	case "sources sought":
		return hexColor("#D97706")
	case "award notice":
		return hexColor("#16A34A")
	case "special notice":
		return hexColor("#64748B")
	case "combined synopsis/solicitation":
		return hexColor("#1E40AF")
	default:
		return hexColor("#64748B")
	}
}

func setAsideBadgeColor(code string) color.Color {
	switch strings.ToUpper(code) {
	case "SBA", "SBP":
		return hexColor("#2563EB")
	case "8A", "8AN":
		return hexColor("#7C3AED")
	case "HZC", "HZS":
		return hexColor("#059669")
	case "SDVOSBC", "SDVOSBS":
		return hexColor("#9F1239")
	case "WOSB", "WOSBSS", "EDWOSB":
		return hexColor("#0D9488")
	default:
		return hexColor("#475569")
	}
}

func shortenSetAside(code, desc string) string {
	switch strings.ToUpper(code) {
	case "SBA", "SBP":
		return "Small Business"
	case "8A", "8AN":
		return "8(a)"
	case "HZC", "HZS":
		return "HUBZone"
	case "SDVOSBC", "SDVOSBS":
		return "SDVOSB"
	case "WOSB", "WOSBSS":
		return "WOSB"
	case "EDWOSB":
		return "EDWOSB"
	case "VSA", "VSB":
		return "VOSB"
	}
	if desc == "" {
		return ""
	}
	lower := strings.ToLower(desc)
	if strings.Contains(lower, "no set aside") || strings.Contains(lower, "unrestricted") {
		return "Unrestricted"
	}
	if len(desc) > 20 {
		return desc[:20]
	}
	return desc
}

func formatDeadline(t time.Time) (string, color.Color) {
	daysUntil := int(time.Until(t).Hours() / 24)
	dateStr := t.Format("Jan 2, 2006")

	if daysUntil < 0 {
		return dateStr + " (Closed)", subtleColor
	}
	if daysUntil < 7 {
		return "CLOSING SOON · " + dateStr, redColor
	}
	if daysUntil < 30 {
		return dateStr, amberColor
	}
	return dateStr, hexColor("#E2E8F0")
}

func hexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}


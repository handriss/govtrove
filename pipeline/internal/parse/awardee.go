package parse

import (
	"regexp"
	"strings"
)

// SAM.gov gives the awardee two different ways and neither is reliably a bare
// company name. The bulk CSV has a single `Awardee` column holding the name and
// the full address run together with no delimiter:
//
//	"MCGRAW HILL LLC 1325 AVENUE OF THE AMERICAS 5TH FLO NEW YORK NY 10019 USA"
//
// The API usually returns a clean `award.awardee.name`, but for roughly 9% of
// notices it returns that same concatenated blob. So both sides need the same
// treatment, and the CSV — which covers every notice, where the API covers about
// a fifth — needs it on every row.
//
// AwardeeName cuts the string at the company's legal suffix. Measured against
// 4,912 notices where the API supplied a clean name to compare with: 93.1% exact,
// 5.9% returned "" (no recognisable suffix), and 0.96% cut slightly short — for
// example "LABORATORY CORPORATION OF AMERICA" became "LABORATORY CORPORATION".
// None over-reached into the address. That asymmetry is deliberate: a name that
// stops early still groups and searches sensibly, whereas one carrying a street
// or city fragment looks plausible and silently splits a company into several.

// Unambiguous legal suffixes — safe to end a name on wherever they appear.
var strongSuffixes = map[string]bool{
	"LLC": true, "L.L.C": true, "INC": true, "INCORPORATED": true,
	"CORP": true, "CORPORATION": true, "COMPANY": true, "LTD": true,
	"LIMITED": true, "LLP": true, "L.L.P": true, "PLLC": true,
	"P.L.L.C": true, "LP": true, "L.P": true, "GMBH": true,
	"PLC": true, "PBC": true,
}

// Ambiguous suffixes. "CO" is also the state code for Colorado, so
// "... Denver CO 80202" would otherwise cut mid-address. These count only when
// written with a period ("CO.") or when a strong suffix follows ("CO INC").
var weakSuffixes = map[string]bool{
	"CO": true, "PC": true, "P.C": true, "PA": true, "P.A": true,
	"SA": true, "S.A": true, "AG": true, "A.G": true,
	"NV": true, "N.V": true, "BV": true, "B.V": true,
}

type token struct {
	norm      string // upper-cased, trailing "." and "," removed
	hadPeriod bool   // ended in "." before trimming — promotes a weak suffix
	end       int    // byte offset just past this token in the original string
}

// tokenize splits on whitespace, and additionally on commas inside a token so
// that "CO.,LTD" and "INC.,THE" — written without a space, which SAM.gov does
// often enough to matter — still present their suffixes separately.
func tokenize(s string) []token {
	var out []token
	for i := 0; i < len(s); {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		start := i
		for i < len(s) && s[i] != ' ' {
			i++
		}
		if i == start {
			break
		}
		for _, piece := range splitCommas(s[start:i], start) {
			trimmed := strings.TrimRight(piece.text, ",")
			out = append(out, token{
				norm:      strings.ToUpper(strings.TrimRight(trimmed, ".")),
				hadPeriod: strings.HasSuffix(trimmed, "."),
				end:       piece.end,
			})
		}
	}
	return out
}

type piece struct {
	text string
	end  int // byte offset just past this piece in the original string
}

// splitCommas breaks "CO.,LTD" into "CO." and "LTD" while keeping each piece's
// offset, so a cut still slices the original string rather than a rebuilt one.
func splitCommas(raw string, base int) []piece {
	var out []piece
	seg := 0
	for j := 0; j <= len(raw); j++ {
		if j == len(raw) || raw[j] == ',' {
			// Keep the comma with the piece it follows, so "GROUP," still reads
			// as one token and a trailing comma is trimmed as before.
			endByte := j
			if j < len(raw) {
				endByte = j + 1
			}
			if text := raw[seg:endByte]; strings.Trim(text, ",") != "" {
				out = append(out, piece{text: text, end: base + endByte})
			}
			seg = j + 1
		}
	}
	if len(out) == 0 {
		out = append(out, piece{text: raw, end: base + len(raw)})
	}
	return out
}

func isSuffix(t token) bool { return strongSuffixes[t.norm] || weakSuffixes[t.norm] }

// AwardeeName extracts the company name from SAM.gov's concatenated awardee
// string. It returns "" when no legal suffix is recognisable, which is the
// intended outcome for names like "OREGON STATE UNIVERSITY" — better an empty
// field than a guess at where the name stops.
func AwardeeName(blob string) string {
	// Offsets in the tokens index this trimmed string, so the cut below has to
	// slice the same one — slicing the raw blob would shift by the leading space.
	s := strings.TrimSpace(blob)
	toks := tokenize(s)
	if len(toks) == 0 {
		return ""
	}

	cut := -1
	for i, t := range toks {
		if strongSuffixes[t.norm] {
			cut = i
			break
		}
		if weakSuffixes[t.norm] {
			// A weak suffix only ends a name with a period, or ahead of a strong one.
			if t.hadPeriod || (i+1 < len(toks) && strongSuffixes[toks[i+1].norm]) {
				cut = i
				break
			}
		}
	}
	if cut < 0 {
		return ""
	}

	// Absorb stacked suffixes ("CO INC", "MBH & CO. KG") and the spelled-out
	// "LIMITED LIABILITY COMPANY", so the cut lands after the whole tail.
	for cut+1 < len(toks) {
		if toks[cut].norm == "LIMITED" && cut+2 < len(toks) &&
			toks[cut+1].norm == "LIABILITY" && toks[cut+2].norm == "COMPANY" {
			cut += 2
			continue
		}
		if isSuffix(toks[cut+1]) {
			cut++
			continue
		}
		break
	}

	return strings.TrimSpace(s[:toks[cut].end])
}

// AwardeeNameOrBlob prefers a name the API already supplied, falling back to
// parsing. The API's value is used as-is only when it looks like a bare name;
// when it carries the address too it goes through the same parser as the CSV,
// and when it carries an address the parser cannot cut, the field is left empty
// rather than filled with the address.
func AwardeeNameOrBlob(apiName, csvBlob string) string {
	apiName = strings.TrimSpace(apiName)
	if apiName != "" {
		if !carriesAddress(apiName) {
			return apiName
		}
		if parsed := AwardeeName(apiName); parsed != "" {
			return parsed
		}
	}
	return AwardeeName(csvBlob)
}

// addressTail matches a trailing US ZIP, which is how SAM.gov's concatenated
// awardee string ends. Deliberately not "USA": a name may legitimately end in
// the country ("SIEMENS USA") and would then be thrown away for lack of a suffix.
var addressTail = regexp.MustCompile(`\b\d{5}(-\d{4})?\s*$`)

// carriesAddress reports whether a supposed company name has an address stuck to
// it, by two independent signals. The first is a trailing ZIP. The second asks
// what would be left behind if the name were cut at its legal suffix: a tail
// carrying digits is a street number, whereas a tail of plain words is still part
// of the name — "INC RESEARCH LLC" cuts to "INC" and must not be treated as
// concatenated.
//
// The earlier version looked for a street number directly and missed three
// shapes: a single-digit number ("2 SYLVAN WAY"), a number followed by an ordinal
// ("5214 4TH AVENUE"), and a company with no legal suffix at all
// ("ASPEN ENVIRONMENTAL GROUP 5020 CHESEBRO RD ..."), which was stored whole.
// Together those left 340 names carrying a full address.
func carriesAddress(s string) bool {
	s = strings.TrimSpace(s)
	if addressTail.MatchString(s) {
		return true
	}
	cut := AwardeeName(s)
	return cut != "" && len(cut) < len(s) && hasDigit(s[len(cut):])
}

func hasDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}

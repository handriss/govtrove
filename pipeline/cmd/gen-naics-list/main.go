// Command gen-naics-list renders the crawlable NAICS code directory that
// /naics-code-finder/ embeds.
//
// The PSC finder got this treatment in 5874322 and sits at position ~14; the NAICS
// finder was left as a bare search widget at ~30, linking to none of its 640+ leaf
// pages. Those pages were orphaned. This exists so the list can be regenerated when
// the NAICS table changes, rather than being a one-off paste nobody can reproduce.
//
// Every code in naics.Codes() gets a page from the seo-pages Lambda regardless of
// opportunity count, so linking all of them cannot 404.
//
// Usage: go run ./cmd/gen-naics-list > fragment.html
package main

import (
	"fmt"
	"html"
	"os"
	"sort"
	"strings"

	"github.com/handriss/govtrove/pipeline/internal/naics"
)

func main() {
	codes := naics.Codes()
	sectors := naics.Sectors()

	bySector := map[string][]string{}
	for code := range codes {
		sec := naics.SectorOf(code)
		if sectors[sec] == "" {
			continue
		}
		bySector[sec] = append(bySector[sec], code)
	}

	secKeys := make([]string, 0, len(bySector))
	for sec := range bySector {
		secKeys = append(secKeys, sec)
	}
	sort.Strings(secKeys)

	var b strings.Builder
	total := 0
	for _, sec := range secKeys {
		list := bySector[sec]
		sort.Strings(list)
		total += len(list)

		fmt.Fprintf(&b, "                <h3 id=\"naics-%s\"><a href=\"/contracts/naics/sector/%s.html\">NAICS %s &mdash; %s</a></h3>\n",
			sec, sec, sec, html.EscapeString(sectors[sec]))
		b.WriteString("                <table class=\"naics-directory\"><tbody>\n")
		for _, code := range list {
			fmt.Fprintf(&b, "                    <tr><td><a href=\"/contracts/naics/%s.html\">%s</a></td><td>%s</td></tr>\n",
				code, code, html.EscapeString(codes[code]))
		}
		b.WriteString("                </tbody></table>\n")
	}

	fmt.Fprintf(os.Stderr, "sectors=%d codes=%d\n", len(secKeys), total)
	os.Stdout.WriteString(b.String())
}

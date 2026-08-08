package searchrescue

import "strings"

type parsedQuery struct {
	phrases  []string
	terms    []string
	excluded []string
	hasOR    bool
}

// parseQuery mirrors the search's token semantics: quoted spans are literal
// phrases, OR is an operator, -word excludes, everything else is an ANDed term.
func parseQuery(q string) parsedQuery {
	var pq parsedQuery
	var token strings.Builder
	inQuote := false

	flush := func() {
		t := token.String()
		token.Reset()
		if t == "" {
			return
		}
		switch {
		case inQuote:
			pq.phrases = append(pq.phrases, t)
		case t == "OR":
			pq.hasOR = true
		case strings.HasPrefix(t, "-") && len(t) > 1:
			pq.excluded = append(pq.excluded, t[1:])
		default:
			pq.terms = append(pq.terms, t)
		}
	}

	for _, r := range q {
		switch {
		case r == '"':
			flush()
			inQuote = !inQuote
		case r == ' ' || r == '\t' || r == '\n':
			if inQuote {
				token.WriteRune(r)
			} else {
				flush()
			}
		default:
			token.WriteRune(r)
		}
	}
	inQuote = false
	flush()
	return pq
}

func stripQuotes(q string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(q, `"`, " ")), " ")
}

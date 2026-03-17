package reconcile

import (
	"strings"

	"github.com/biter777/countries"
)

var usStateLookup = map[string]string{
	"AL": "Alabama",
	"AK": "Alaska",
	"AZ": "Arizona",
	"AR": "Arkansas",
	"CA": "California",
	"CO": "Colorado",
	"CT": "Connecticut",
	"DE": "Delaware",
	"FL": "Florida",
	"GA": "Georgia",
	"HI": "Hawaii",
	"ID": "Idaho",
	"IL": "Illinois",
	"IN": "Indiana",
	"IA": "Iowa",
	"KS": "Kansas",
	"KY": "Kentucky",
	"LA": "Louisiana",
	"ME": "Maine",
	"MD": "Maryland",
	"MA": "Massachusetts",
	"MI": "Michigan",
	"MN": "Minnesota",
	"MS": "Mississippi",
	"MO": "Missouri",
	"MT": "Montana",
	"NE": "Nebraska",
	"NV": "Nevada",
	"NH": "New Hampshire",
	"NJ": "New Jersey",
	"NM": "New Mexico",
	"NY": "New York",
	"NC": "North Carolina",
	"ND": "North Dakota",
	"OH": "Ohio",
	"OK": "Oklahoma",
	"OR": "Oregon",
	"PA": "Pennsylvania",
	"RI": "Rhode Island",
	"SC": "South Carolina",
	"SD": "South Dakota",
	"TN": "Tennessee",
	"TX": "Texas",
	"UT": "Utah",
	"VT": "Vermont",
	"VA": "Virginia",
	"WA": "Washington",
	"WV": "West Virginia",
	"WI": "Wisconsin",
	"WY": "Wyoming",
	"DC": "District of Columbia",
	"PR": "Puerto Rico",
	"GU": "Guam",
	"VI": "U.S. Virgin Islands",
	"AS": "American Samoa",
	"MP": "Northern Mariana Islands",
	"UM": "U.S. Minor Outlying Islands",
}

// LookupStateName returns the full state name for a 2-letter US state code.
// Returns the input unchanged if not found (so API records with full names pass through).
func LookupStateName(code string) string {
	if code == "" {
		return ""
	}
	if name, ok := usStateLookup[strings.ToUpper(code)]; ok {
		return name
	}
	return code
}

// LookupCountryName returns the full country name for an ISO 3166-1 alpha-3 code.
// Uses the biter777/countries library instead of a hardcoded map.
// Returns the input unchanged if not recognized.
func LookupCountryName(code string) string {
	if code == "" {
		return ""
	}
	c := countries.ByName(strings.ToUpper(code))
	if c == countries.Unknown {
		return code
	}
	return c.String()
}

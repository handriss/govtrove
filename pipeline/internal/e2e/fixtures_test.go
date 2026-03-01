package e2e_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
)

// CSV headers matching the SAM.gov Contract Opportunities format.
// Both ExtractSnapCSVRow and reconcile.FromCSV read from these headers.
//
//	Index:  0         1     2      3     4         5           6                7            8
//	Header: NoticeId, Sol#, Title, Type, BaseType, PostedDate, ResponseDeadLine, ArchiveDate, ArchiveType,
//	Index:  9             10       11         12                  13
//	Header: SetASideCode, SetASide, NaicsCode, ClassificationCode, Active,
//	Index:  14                      15        16      17    18          19
//	Header: Department/Ind.Agency, Sub-Tier, Office, CGAC, FPDS Code, AAC Code,
//	Index:  20           21         22      23
//	Header: AwardNumber, AwardDate, Award$, Awardee,
//	Index:  24            25                26
//	Header: Description, OrganizationType, Link,
//	Index:  27                    28                       29                    30                    31
//	Header: PrimaryContactTitle, PrimaryContactFullname, PrimaryContactEmail, PrimaryContactPhone, PrimaryContactFax,
//	Index:  32                      33                         34                      35                      36
//	Header: SecondaryContactTitle, SecondaryContactFullname, SecondaryContactEmail, SecondaryContactPhone, SecondaryContactFax,
//	Index:  37                38        39        40      41
//	Header: PopStreetAddress, PopCity, PopState, PopZip, PopCountry,
//	Index:  42    43     44       45
//	Header: City, State, ZipCode, CountryCode
const csvHeaders = "NoticeId,Sol#,Title,Type,BaseType,PostedDate,ResponseDeadLine,ArchiveDate,ArchiveType,SetASideCode,SetASide,NaicsCode,ClassificationCode,Active," +
	"Department/Ind.Agency,Sub-Tier,Office,CGAC,FPDS Code,AAC Code," +
	"AwardNumber,AwardDate,Award$,Awardee," +
	"Description,OrganizationType,Link," +
	"PrimaryContactTitle,PrimaryContactFullname,PrimaryContactEmail,PrimaryContactPhone,PrimaryContactFax," +
	"SecondaryContactTitle,SecondaryContactFullname,SecondaryContactEmail,SecondaryContactPhone,SecondaryContactFax," +
	"PopStreetAddress,PopCity,PopState,PopZip,PopCountry," +
	"City,State,ZipCode,CountryCode"

// row builds a 46-column CSV row from a map of column index → value.
// Unspecified columns are empty strings.
func row(fields map[int]string) string {
	cols := make([]string, 46)
	for i, v := range fields {
		cols[i] = v
	}
	return strings.Join(cols, ",")
}

// Run 1: baseline — 8 active records
var activeCSVRun1 = csvHeaders + "\n" +
	strings.Join([]string{
		// HAPPY-001: fully populated solicitation
		row(map[int]string{
			0: "HAPPY-001", 1: "SOL-001", 2: "Test Solicitation", 3: "Solicitation", 4: "Solicitation",
			5: "01/15/2026", 6: "03/01/2026", 7: "06/01/2026", 8: "auto",
			9: "SBA", 10: "Total Small Business", 11: "541511", 12: "R425", 13: "Yes",
			14: "Department of Defense", 15: "Navy", 16: "NAVSEA", 17: "097", 18: "N00024", 19: "AAC001",
			24: "A test solicitation description", 25: "DoD/Military", 26: "https://sam.gov/opp/HAPPY-001",
			27: "Mr.", 28: "John Doe", 29: "john@navy.mil", 30: "555-0100", 31: "555-0101",
			32: "Ms.", 33: "Jane Smith", 34: "jane@navy.mil", 35: "555-0200", 36: "555-0201",
			37: "123 Main St", 38: "Arlington", 39: "VA", 40: "22201", 41: "US",
			42: "Washington", 43: "DC", 44: "20001", 45: "US",
		}),
		// HAPPY-002: award notice
		row(map[int]string{
			0: "HAPPY-002", 1: "SOL-002", 2: "Test Award Notice", 3: "Award", 4: "Award",
			5: "12/01/2025", 13: "No",
			14: "Department of Defense", 15: "Army", 16: "Army Contracting", 17: "021", 18: "W91CRB",
			20: "W31P4Q-25-C-0001", 21: "02/01/2026", 22: "50000", 23: "ACME Corp Arlington VA 22201 US",
			24: "An award description", 26: "https://sam.gov/opp/HAPPY-002",
		}),
		// SENTINEL-001: sentinel archive date (1969-12-31)
		row(map[int]string{
			0: "SENTINEL-001", 1: "SOL-SENT", 2: "Sentinel Test", 3: "Solicitation", 4: "Solicitation",
			5: "01/10/2026", 7: "12/31/1969", 8: "auto", 11: "541330", 13: "Yes",
			14: "Dept of Energy", 17: "019",
		}),
		// BADDATE-001: unparseable archive date
		row(map[int]string{
			0: "BADDATE-001", 1: "SOL-BAD", 2: "Bad Date Record", 3: "Solicitation", 4: "Solicitation",
			7: "not-a-date", 13: "Yes",
			14: "GSA", 17: "047",
		}),
		// MINIMAL-001: sparse record
		row(map[int]string{
			0: "MINIMAL-001", 2: "Minimal Record", 3: "Solicitation",
		}),
		// WILL-CHANGE: modified in run 2
		row(map[int]string{
			0: "WILL-CHANGE", 1: "SOL-CHG", 2: "Original Title", 3: "Solicitation", 4: "Solicitation",
			5: "01/05/2026", 13: "Yes",
			14: "DHS", 17: "070",
			22: "1000",
		}),
		// WILL-DISAPPEAR: removed in run 2
		row(map[int]string{
			0: "WILL-DISAPPEAR", 1: "SOL-DISAP", 2: "Disappearing Record", 3: "Solicitation", 4: "Solicitation",
			5: "01/01/2026", 7: "03/01/2026", 8: "auto", 11: "541511", 13: "Yes",
			14: "DoD", 17: "097",
		}),
		// WILL-GLITCH: removed in run 2, returns in run 3
		row(map[int]string{
			0: "WILL-GLITCH", 1: "SOL-GLITCH", 2: "Glitch Record", 3: "Solicitation", 4: "Solicitation",
			5: "01/02/2026", 13: "Yes",
			14: "DoD", 17: "097",
		}),
	}, "\n")

// Run 2: WILL-CHANGE modified, BRAND-NEW added, WILL-DISAPPEAR and WILL-GLITCH absent
var activeCSVRun2 = csvHeaders + "\n" +
	strings.Join([]string{
		row(map[int]string{
			0: "HAPPY-001", 1: "SOL-001", 2: "Test Solicitation", 3: "Solicitation", 4: "Solicitation",
			5: "01/15/2026", 6: "03/01/2026", 7: "06/01/2026", 8: "auto",
			9: "SBA", 10: "Total Small Business", 11: "541511", 12: "R425", 13: "Yes",
			14: "Department of Defense", 15: "Navy", 16: "NAVSEA", 17: "097", 18: "N00024", 19: "AAC001",
			24: "A test solicitation description", 25: "DoD/Military", 26: "https://sam.gov/opp/HAPPY-001",
			27: "Mr.", 28: "John Doe", 29: "john@navy.mil", 30: "555-0100", 31: "555-0101",
			32: "Ms.", 33: "Jane Smith", 34: "jane@navy.mil", 35: "555-0200", 36: "555-0201",
			37: "123 Main St", 38: "Arlington", 39: "VA", 40: "22201", 41: "US",
			42: "Washington", 43: "DC", 44: "20001", 45: "US",
		}),
		row(map[int]string{
			0: "HAPPY-002", 1: "SOL-002", 2: "Test Award Notice", 3: "Award", 4: "Award",
			5: "12/01/2025", 13: "No",
			14: "Department of Defense", 15: "Army", 16: "Army Contracting", 17: "021", 18: "W91CRB",
			20: "W31P4Q-25-C-0001", 21: "02/01/2026", 22: "50000", 23: "ACME Corp Arlington VA 22201 US",
			24: "An award description", 26: "https://sam.gov/opp/HAPPY-002",
		}),
		row(map[int]string{
			0: "SENTINEL-001", 1: "SOL-SENT", 2: "Sentinel Test", 3: "Solicitation", 4: "Solicitation",
			5: "01/10/2026", 7: "12/31/1969", 8: "auto", 11: "541330", 13: "Yes",
			14: "Dept of Energy", 17: "019",
		}),
		row(map[int]string{
			0: "BADDATE-001", 1: "SOL-BAD", 2: "Bad Date Record", 3: "Solicitation", 4: "Solicitation",
			7: "not-a-date", 13: "Yes",
			14: "GSA", 17: "047",
		}),
		row(map[int]string{
			0: "MINIMAL-001", 2: "Minimal Record", 3: "Solicitation",
		}),
		// WILL-CHANGE: Title and Award$ modified
		row(map[int]string{
			0: "WILL-CHANGE", 1: "SOL-CHG", 2: "Updated Title", 3: "Solicitation", 4: "Solicitation",
			5: "01/05/2026", 13: "Yes",
			14: "DHS", 17: "070",
			22: "2000",
		}),
		// BRAND-NEW: first appearance
		row(map[int]string{
			0: "BRAND-NEW", 1: "SOL-NEW", 2: "Brand New Opportunity", 3: "Solicitation", 4: "Solicitation",
			5: "02/15/2026", 6: "04/01/2026", 13: "Yes",
			14: "NASA", 17: "080",
		}),
	}, "\n")

// Run 3: same as run 2 + WILL-GLITCH returns
var activeCSVRun3 = activeCSVRun2 + "\n" +
	row(map[int]string{
		0: "WILL-GLITCH", 1: "SOL-GLITCH", 2: "Glitch Record", 3: "Solicitation", 4: "Solicitation",
		5: "01/02/2026", 13: "Yes",
		14: "DoD", 17: "097",
	})

func gzipCSV(csv string) io.ReadCloser {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte(csv))
	gz.Close()
	return io.NopCloser(&buf)
}


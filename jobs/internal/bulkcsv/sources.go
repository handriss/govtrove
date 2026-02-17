package bulkcsv

import "github.com/handriss/govtrove/jobs/internal/samgov"

type Source struct {
	Key           string
	URL           string
	S3Prefix      string
	UseRangeProbe bool // true = ETag probe via 1-byte range request (for redirect-based downloads)
}

var Sources = []Source{
	{
		Key:      "active",
		URL:      samgov.FullCSVURL,
		S3Prefix: "raw/csv",
	},
	{
		Key:           "archived_fy2026",
		URL:            "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2026_archived_opportunities.csv",
		S3Prefix:       "raw/archived-csv/FY2026",
		UseRangeProbe: true,
	},
}

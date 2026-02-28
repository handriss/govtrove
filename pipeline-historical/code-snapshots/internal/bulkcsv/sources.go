package bulkcsv

import "github.com/handriss/govtrove/pipeline/internal/samgov"

type SourceType string

const (
	SourceTypeActive   SourceType = "active"
	SourceTypeArchived SourceType = "archived"
)

type Source struct {
	Key           string
	Type          SourceType
	URL           string
	S3Prefix      string
	UseRangeProbe bool // true = ETag probe via 1-byte range request (for redirect-based downloads)
}

var Sources = []Source{
	{
		Key:      "active",
		Type:     SourceTypeActive,
		URL:      samgov.FullCSVURL,
		S3Prefix: "raw/csv",
	},
	{
		Key:           "archived_fy2025",
		Type:          SourceTypeArchived,
		URL:           "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2025_archived_opportunities.csv",
		S3Prefix:      "raw/archived-csv/FY2025",
		UseRangeProbe: true,
	},
	{
		Key:           "archived_fy2026",
		Type:          SourceTypeArchived,
		URL:           "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/Archived%20Data/FY2026_archived_opportunities.csv",
		S3Prefix:      "raw/archived-csv/FY2026",
		UseRangeProbe: true,
	},
}

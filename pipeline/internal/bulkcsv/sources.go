package bulkcsv

import "github.com/handriss/govtrove/pipeline/internal/samgov"

type SourceType string

const (
	SourceTypeActive SourceType = "active"
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
}

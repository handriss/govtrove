package samgov

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/handriss/govtrove/ingestion/internal/database"
	"golang.org/x/text/encoding/charmap"
)

const (
	FullCSVURL = "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/datagov/ContractOpportunitiesFullCSV.csv"
)

type CSVClient struct {
	httpClient     *TrackedHTTPClient
	logger         *slog.Logger
	minPostedDate  *time.Time
	skippedByDate  int
}

func NewCSVClient(db *database.DB, logger *slog.Logger) *CSVClient {
	return &CSVClient{
		httpClient: NewTrackedHTTPClient(db, logger),
		logger:     logger,
	}
}

func (c *CSVClient) SetIngestionRunID(runID int) {
	c.httpClient.SetIngestionRunID(runID)
}

func (c *CSVClient) SetMinPostedDate(d *time.Time) {
	c.minPostedDate = d
}

func (c *CSVClient) SkippedByDate() int {
	return c.skippedByDate
}

type FetchResult struct {
	Opportunities []*database.Opportunity
	TotalRows     int
	ParsedRows    int
	Errors        int
	NotModified   bool   // true if server returned 304 Not Modified
	ETag          string // ETag header from response
	LastModified  string // Last-Modified header from response
}

func (c *CSVClient) FetchOpportunities(ctx context.Context, limit int) (*FetchResult, error) {
	return c.FetchOpportunitiesConditional(ctx, limit, "", "")
}

// FetchOpportunitiesConditional fetches the CSV with optional conditional headers.
// If etag or lastModified are provided, the request includes If-None-Match and/or
// If-Modified-Since headers. Returns NotModified=true if server returns 304.
func (c *CSVClient) FetchOpportunitiesConditional(ctx context.Context, limit int, etag, lastModified string) (*FetchResult, error) {
	c.logger.Info("starting CSV download", "url", FullCSVURL, "has_etag", etag != "", "has_last_modified", lastModified != "")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, FullCSVURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		c.logger.Info("CSV not modified (304), skipping download")
		return &FetchResult{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Info("CSV download started, parsing stream", "response_time_ms", resp.ResponseTimeMs)

	result, err := c.parseCSV(resp.Body, limit)
	if err != nil {
		return nil, err
	}

	// Capture response headers for caching
	result.ETag = resp.Header.Get("ETag")
	result.LastModified = resp.Header.Get("Last-Modified")

	return result, nil
}

func (c *CSVClient) parseCSV(r io.Reader, limit int) (*FetchResult, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	headerIndex := make(map[string]int)
	for i, h := range headers {
		headerIndex[strings.TrimSpace(h)] = i
	}

	c.skippedByDate = 0
	result := &FetchResult{
		Opportunities: make([]*database.Opportunity, 0, limit),
	}

	startTime := time.Now()
	const progressInterval = 10000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.logger.Warn("failed to read CSV row", "error", err, "row", result.TotalRows+1)
			result.Errors++
			result.TotalRows++
			continue
		}

		result.TotalRows++

		// Filter by posted date BEFORE allocating the full Opportunity struct.
		// This keeps memory bounded to only the records we'll actually use.
		if c.minPostedDate != nil {
			if idx, ok := headerIndex["PostedDate"]; ok && idx < len(record) {
				if d := parseDate(strings.TrimSpace(record[idx])); d != nil && d.Before(*c.minPostedDate) {
					c.skippedByDate++
					continue
				}
			}
		}

		opp, err := c.mapRowToOpportunity(record, headerIndex)
		if err != nil {
			c.logger.Debug("failed to map row", "error", err, "row", result.TotalRows)
			result.Errors++
			continue
		}

		result.Opportunities = append(result.Opportunities, opp)
		result.ParsedRows++

		if result.ParsedRows%progressInterval == 0 {
			elapsed := time.Since(startTime)
			rate := float64(result.ParsedRows) / elapsed.Seconds()
			c.logger.Info("CSV parsing progress",
				"parsed", result.ParsedRows,
				"skipped", c.skippedByDate,
				"errors", result.Errors,
				"elapsed", elapsed.Round(time.Second),
				"rate", fmt.Sprintf("%.0f/sec", rate),
			)
		}

		if limit > 0 && result.ParsedRows >= limit {
			c.logger.Info("reached record limit", "limit", limit)
			break
		}
	}

	c.logger.Info("CSV parsing complete",
		"total_rows", result.TotalRows,
		"parsed_rows", result.ParsedRows,
		"skipped_by_date", c.skippedByDate,
		"errors", result.Errors,
	)

	return result, nil
}

func (c *CSVClient) mapRowToOpportunity(record []string, headerIndex map[string]int) (*database.Opportunity, error) {
	getValue := func(columnName string) string {
		if idx, ok := headerIndex[columnName]; ok && idx < len(record) {
			return sanitizeToUTF8(strings.TrimSpace(record[idx]))
		}
		return ""
	}

	noticeID := getValue("NoticeId")
	if noticeID == "" {
		return nil, fmt.Errorf("missing NoticeId")
	}

	title := getValue("Title")
	if title == "" {
		return nil, fmt.Errorf("missing Title for NoticeId %s", noticeID)
	}

	dept := getValue("Department/Ind.Agency")
	subTier := getValue("Sub-Tier")
	office := getValue("Office")

	var fullParentPathName string
	if dept != "" {
		parts := []string{dept}
		if subTier != "" {
			parts = append(parts, subTier)
		}
		if office != "" {
			parts = append(parts, office)
		}
		fullParentPathName = strings.Join(parts, ".")
	}

	opp := &database.Opportunity{
		NoticeID:            noticeID,
		SolicitationNumber:  getValue("Sol#"),
		Title:               title,
		Description:         getValue("Description"),
		UILink:              getValue("Link"),
		AdditionalInfoLink:  getValue("AdditionalInfoLink"),
		Type:                getValue("Type"),
		BaseType:            getValue("BaseType"),
		SetAsideCode:        getValue("SetASideCode"),
		SetAsideDescription: getValue("SetASide"),
		NAICSCode:           getValue("NaicsCode"),
		ClassificationCode:  getValue("ClassificationCode"),
		OrganizationType:    getValue("OrganizationType"),
		ArchiveType:         getValue("ArchiveType"),
		FullParentPathName:  fullParentPathName,
		Active:              parseActive(getValue("Active")),

		Department: dept,
		SubTier:    subTier,
		Office:     office,
		CGAC:       getValue("CGAC"),
		FPDSCode:   getValue("FPDS Code"),
		AACCode:    getValue("AAC Code"),

		PopStreetAddress: getValue("PopStreetAddress"),
		PopCity:          getValue("PopCity"),
		PopStateCode:     getValue("PopState"),
		PopZip:           getValue("PopZip"),
		PopCountryCode:   getValue("PopCountry"),

		OfficeCity:        getValue("City"),
		OfficeState:       getValue("State"),
		OfficeZip:         getValue("ZipCode"),
		OfficeCountryCode: getValue("CountryCode"),

		AwardNumber: getValue("AwardNumber"),
		AwardeeName: getValue("Awardee"),

		PrimaryContactTitle:    getValue("PrimaryContactTitle"),
		PrimaryContactFullname: getValue("PrimaryContactFullname"),
		PrimaryContactEmail:    getValue("PrimaryContactEmail"),
		PrimaryContactPhone:    getValue("PrimaryContactPhone"),
		PrimaryContactFax:      getValue("PrimaryContactFax"),

		SecondaryContactTitle:    getValue("SecondaryContactTitle"),
		SecondaryContactFullname: getValue("SecondaryContactFullname"),
		SecondaryContactEmail:    getValue("SecondaryContactEmail"),
		SecondaryContactPhone:    getValue("SecondaryContactPhone"),
		SecondaryContactFax:      getValue("SecondaryContactFax"),
	}

	opp.PostedDate = parseDate(getValue("PostedDate"))
	opp.ResponseDeadline = parseDate(getValue("ResponseDeadLine"))
	opp.ArchiveDate = parseDate(getValue("ArchiveDate"))
	opp.AwardDate = parseDateOnly(getValue("AwardDate"))
	opp.AwardAmount = parseAmount(getValue("Award$"))

	rawData := make(map[string]string)
	for header, idx := range headerIndex {
		if idx < len(record) && record[idx] != "" {
			rawData[header] = sanitizeToUTF8(record[idx])
		}
	}
	rawJSON, _ := json.Marshal(rawData)
	opp.RawJSON = rawJSON

	return opp, nil
}

// sanitizeToUTF8 converts Windows-1252 encoded text to valid UTF-8.
// SAM.gov CSV files often contain Windows-1252 characters (smart quotes, etc.)
// that are invalid in UTF-8 and cause PostgreSQL insertion failures.
func sanitizeToUTF8(s string) string {
	if s == "" {
		return s
	}
	// Try to decode as Windows-1252 and convert to UTF-8
	decoder := charmap.Windows1252.NewDecoder()
	result, err := decoder.String(s)
	if err != nil {
		// Fallback: strip any remaining invalid UTF-8 bytes
		return strings.ToValidUTF8(s, "")
	}
	return result
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}

	formats := []string{
		// SAM.gov CSV format with milliseconds and timezone (e.g., "2026-01-23 12:59:11.178-05")
		"2006-01-02 15:04:05.000-07",
		"2006-01-02 15:04:05.00-07",
		"2006-01-02 15:04:05.0-07",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05.000-0700",
		"2006-01-02 15:04:05-0700",
		// Other common formats
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006 15:04",
		"01/02/2006",
		"0102/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

func parseDateOnly(s string) *time.Time {
	if s == "" {
		return nil
	}
	return parseDate(s)
}

func parseActive(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "yes" || s == "true" || s == "1"
}

func parseAmount(s string) *float64 {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.TrimSpace(s)

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return &f
	}
	return nil
}

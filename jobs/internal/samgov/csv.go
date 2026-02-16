package samgov

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/handriss/govtrove/jobs/internal/database"
	"github.com/handriss/govtrove/jobs/internal/parse"
	"golang.org/x/text/encoding/charmap"
)

const (
	FullCSVURL = "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/datagov/ContractOpportunitiesFullCSV.csv"
)

type CSVClient struct {
	httpClient *TrackedHTTPClient
	logger     *slog.Logger
}

func NewCSVClient(db *database.DB, logger *slog.Logger) *CSVClient {
	return &CSVClient{
		httpClient: NewTrackedHTTPClient(db, logger),
		logger:     logger,
	}
}

type CSVParseResult struct {
	Rows         []map[string]string
	Headers      []string
	TotalRows    int
	ParsedRows   int
	Errors       int
	NotModified  bool
	ETag         string
	LastModified string
}

func (c *CSVClient) FetchOpportunities(ctx context.Context, limit int) (*CSVParseResult, error) {
	return c.FetchOpportunitiesConditional(ctx, limit, "", "")
}

func (c *CSVClient) FetchOpportunitiesConditional(ctx context.Context, limit int, etag, lastModified string) (*CSVParseResult, error) {
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
		return &CSVParseResult{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Info("CSV download started, parsing stream", "response_time_ms", resp.ResponseTimeMs)

	result, err := c.parseCSV(resp.Body, limit)
	if err != nil {
		return nil, err
	}

	result.ETag = resp.Header.Get("ETag")
	result.LastModified = resp.Header.Get("Last-Modified")

	return result, nil
}

func (c *CSVClient) parseCSV(r io.Reader, limit int) (*CSVParseResult, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	headerIndex := make(map[string]int)
	cleanHeaders := make([]string, len(headers))
	for i, h := range headers {
		clean := strings.TrimSpace(h)
		headerIndex[clean] = i
		cleanHeaders[i] = clean
	}

	result := &CSVParseResult{
		Headers: cleanHeaders,
		Rows:    make([]map[string]string, 0, 100000),
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

		row := make(map[string]string, len(cleanHeaders))
		for i, h := range cleanHeaders {
			if i < len(record) {
				val := sanitizeToUTF8(strings.TrimSpace(record[i]))
				if val != "" {
					row[h] = val
				}
			}
		}

		if row["NoticeId"] == "" {
			result.Errors++
			continue
		}

		result.Rows = append(result.Rows, row)
		result.ParsedRows++

		if result.ParsedRows%progressInterval == 0 {
			elapsed := time.Since(startTime)
			rate := float64(result.ParsedRows) / elapsed.Seconds()
			c.logger.Info("CSV parsing progress",
				"parsed", result.ParsedRows,
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
		"errors", result.Errors,
	)

	return result, nil
}

func sanitizeToUTF8(s string) string {
	if s == "" {
		return s
	}
	decoder := charmap.Windows1252.NewDecoder()
	result, err := decoder.String(s)
	if err != nil {
		return strings.ToValidUTF8(s, "")
	}
	return result
}

func ParseDate(s string) *time.Time    { return parse.Date(s) }
func ParseActive(s string) bool         { return parse.Active(s) }
func ParseAmount(s string) *float64     { return parse.Amount(s) }

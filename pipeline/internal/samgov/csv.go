package samgov

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/parse"
	"golang.org/x/text/encoding/charmap"
)

const (
	FullCSVURL = "https://sam.gov/api/prod/fileextractservices/v1/api/download/Contract%20Opportunities/datagov/ContractOpportunitiesFullCSV.csv"
)

type CSVParseResult struct {
	Rows        []map[string]string
	Headers     []string
	TotalRows   int
	ParsedRows  int
	Errors      int
	NotModified bool
	ETag        string
	LastModified string
}

func ParseCSVFromReader(r io.Reader, limit int, logger *slog.Logger) (*CSVParseResult, error) {
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
			logger.Warn("failed to read CSV row", "error", err, "row", result.TotalRows+1)
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
			logger.Info("CSV parsing progress",
				"parsed", result.ParsedRows,
				"errors", result.Errors,
				"elapsed", elapsed.Round(time.Second),
				"rate", fmt.Sprintf("%.0f/sec", rate),
			)
		}

		if limit > 0 && result.ParsedRows >= limit {
			logger.Info("reached record limit", "limit", limit)
			break
		}
	}

	logger.Info("CSV parsing complete",
		"total_rows", result.TotalRows,
		"parsed_rows", result.ParsedRows,
		"errors", result.Errors,
	)

	return result, nil
}

// ParseCSVStream reads a CSV from r and calls fn for each valid row (as a map).
// Rows missing NoticeId are skipped. If fn returns an error, parsing stops immediately.
func ParseCSVStream(r io.Reader, logger *slog.Logger, fn func(map[string]string) error) (total, parsed, errors int, err error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	cleanHeaders := make([]string, len(headers))
	for i, h := range headers {
		cleanHeaders[i] = strings.TrimSpace(h)
	}

	startTime := time.Now()
	const progressInterval = 10000

	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			logger.Warn("failed to read CSV row", "error", readErr, "row", total+1)
			errors++
			total++
			continue
		}

		total++

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
			errors++
			continue
		}

		if callErr := fn(row); callErr != nil {
			return total, parsed, errors, fmt.Errorf("callback error at row %d: %w", total, callErr)
		}
		parsed++

		if parsed%progressInterval == 0 {
			elapsed := time.Since(startTime)
			rate := float64(parsed) / elapsed.Seconds()
			logger.Info("CSV parsing progress",
				"parsed", parsed,
				"errors", errors,
				"elapsed", elapsed.Round(time.Second),
				"rate", fmt.Sprintf("%.0f/sec", rate),
			)
		}
	}

	logger.Info("CSV streaming complete",
		"total_rows", total,
		"parsed_rows", parsed,
		"errors", errors,
	)

	return total, parsed, errors, nil
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

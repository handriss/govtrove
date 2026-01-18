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

	"github.com/opscout/ingestion/internal/database"
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

func (c *CSVClient) SetIngestionRunID(runID int) {
	c.httpClient.SetIngestionRunID(runID)
}

type FetchResult struct {
	Opportunities []*database.Opportunity
	TotalRows     int
	ParsedRows    int
	Errors        int
}

func (c *CSVClient) FetchOpportunities(ctx context.Context, limit int) (*FetchResult, error) {
	c.logger.Info("starting CSV download", "url", FullCSVURL)

	resp, err := c.httpClient.Get(ctx, FullCSVURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Info("CSV download started, parsing stream", "response_time_ms", resp.ResponseTimeMs)

	return c.parseCSV(resp.Body, limit)
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

	result := &FetchResult{
		Opportunities: make([]*database.Opportunity, 0, limit),
	}

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

		opp, err := c.mapRowToOpportunity(record, headerIndex)
		if err != nil {
			c.logger.Debug("failed to map row", "error", err, "row", result.TotalRows)
			result.Errors++
			continue
		}

		result.Opportunities = append(result.Opportunities, opp)
		result.ParsedRows++

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

func (c *CSVClient) mapRowToOpportunity(record []string, headerIndex map[string]int) (*database.Opportunity, error) {
	getValue := func(columnName string) string {
		if idx, ok := headerIndex[columnName]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
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
			rawData[header] = record[idx]
		}
	}
	rawJSON, _ := json.Marshal(rawData)
	opp.RawJSON = rawJSON

	return opp, nil
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}

	formats := []string{
		"0102/2006",
		"01/02/2006",
		"2006-01-02",
		"01/02/2006 15:04",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
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

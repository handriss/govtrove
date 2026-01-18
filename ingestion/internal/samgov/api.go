package samgov

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/opscout/ingestion/internal/database"
)

const (
	SearchAPIBaseURL = "https://api.sam.gov/prod/opportunities/v2/search"
	DefaultPageLimit = 1000
	PaginationDelay  = 2500 * time.Millisecond
)

type APIClient struct {
	httpClient *TrackedHTTPClient
	logger     *slog.Logger
	apiKey     string
	baseURL    string
}

type APISearchResponse struct {
	TotalRecords      int              `json:"totalRecords"`
	OpportunitiesData []APIOpportunity `json:"opportunitiesData"`
}

type APIOpportunity struct {
	NoticeID           string              `json:"noticeId"`
	Title              string              `json:"title"`
	SolicitationNumber string              `json:"solicitationNumber"`
	FullParentPathName string              `json:"fullParentPathName"`
	FullParentPathCode string              `json:"fullParentPathCode"`
	PostedDate         string              `json:"postedDate"`
	Type               string              `json:"type"`
	BaseType           string              `json:"baseType"`
	ArchiveType        string              `json:"archiveType"`
	ArchiveDate        string              `json:"archiveDate"`
	SetAsideCode       string              `json:"setAsideCode"`
	SetAside           string              `json:"setAside"`
	ResponseDeadline   string              `json:"responseDeadLine"`
	NAICSCode          string              `json:"naicsCode"`
	NAICSCodes         []string            `json:"naicsCodes"`
	ClassificationCode string              `json:"classificationCode"`
	Active             string              `json:"active"`
	OrganizationType   string              `json:"organizationType"`
	UILink             string              `json:"uiLink"`
	OfficeAddress      *OfficeAddress      `json:"officeAddress"`
	PointOfContact     []PointOfContact    `json:"pointOfContact"`
	PlaceOfPerformance *PlaceOfPerformance `json:"placeOfPerformance"`
}

type OfficeAddress struct {
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zipcode"`
	CountryCode string `json:"countryCode"`
}

type PointOfContact struct {
	Type     string `json:"type"`
	FullName string `json:"fullName"`
	Title    string `json:"title"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Fax      string `json:"fax"`
}

type PlaceOfPerformance struct {
	StreetAddress string       `json:"streetAddress"`
	City          *CityInfo    `json:"city"`
	State         *StateInfo   `json:"state"`
	Zip           string       `json:"zip"`
	Country       *CountryInfo `json:"country"`
}

type CityInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type StateInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CountryInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type APIFetchResult struct {
	Opportunities []*database.Opportunity
	TotalRecords  int
	PagesFetched  int
	Errors        int
}

func NewAPIClient(db *database.DB, logger *slog.Logger, apiKey string) *APIClient {
	return &APIClient{
		httpClient: NewTrackedHTTPClient(db, logger),
		logger:     logger,
		apiKey:     apiKey,
		baseURL:    SearchAPIBaseURL,
	}
}

func (c *APIClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *APIClient) SetIngestionRunID(runID int) {
	c.httpClient.SetIngestionRunID(runID)
}

func (c *APIClient) FetchAllOpportunities(ctx context.Context, limit int) (*APIFetchResult, error) {
	result := &APIFetchResult{
		Opportunities: make([]*database.Opportunity, 0),
	}

	offset := 0
	hasMore := true

	for hasMore {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		pageResult, totalRecords, err := c.fetchPage(ctx, offset)
		if err != nil {
			c.logger.Error("failed to fetch page", "offset", offset, "error", err)
			result.Errors++
			break
		}

		if offset == 0 {
			result.TotalRecords = totalRecords
			c.logger.Info("API reports total records", "total", totalRecords)
		}

		result.Opportunities = append(result.Opportunities, pageResult...)
		result.PagesFetched++

		c.logger.Debug("fetched page",
			"offset", offset,
			"count", len(pageResult),
			"total_so_far", len(result.Opportunities),
		)

		if limit > 0 && len(result.Opportunities) >= limit {
			c.logger.Info("reached record limit", "limit", limit)
			result.Opportunities = result.Opportunities[:limit]
			break
		}

		hasMore = len(pageResult) == DefaultPageLimit
		offset += DefaultPageLimit

		if hasMore {
			c.logger.Debug("sleeping before next page", "delay_ms", PaginationDelay.Milliseconds())
			time.Sleep(PaginationDelay)
		}
	}

	c.logger.Info("API fetch complete",
		"total_records", result.TotalRecords,
		"fetched", len(result.Opportunities),
		"pages", result.PagesFetched,
		"errors", result.Errors,
	)

	return result, nil
}

func (c *APIClient) fetchPage(ctx context.Context, offset int) ([]*database.Opportunity, int, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("api_key", c.apiKey)
	q.Set("limit", fmt.Sprintf("%d", DefaultPageLimit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(ctx, u.String())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var searchResp APISearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	opportunities := make([]*database.Opportunity, 0, len(searchResp.OpportunitiesData))
	for _, apiOpp := range searchResp.OpportunitiesData {
		if apiOpp.NoticeID == "" {
			continue
		}
		opp := c.mapAPIToOpportunity(&apiOpp)
		opportunities = append(opportunities, opp)
	}

	return opportunities, searchResp.TotalRecords, nil
}

func (c *APIClient) mapAPIToOpportunity(apiOpp *APIOpportunity) *database.Opportunity {
	opp := &database.Opportunity{
		NoticeID:            apiOpp.NoticeID,
		SolicitationNumber:  apiOpp.SolicitationNumber,
		Title:               apiOpp.Title,
		UILink:              apiOpp.UILink,
		Type:                apiOpp.Type,
		BaseType:            apiOpp.BaseType,
		FullParentPathName:  apiOpp.FullParentPathName,
		FullParentPathCode:  apiOpp.FullParentPathCode,
		SetAsideCode:        apiOpp.SetAsideCode,
		SetAsideDescription: apiOpp.SetAside,
		NAICSCode:           apiOpp.NAICSCode,
		NAICSCodes:          apiOpp.NAICSCodes,
		ClassificationCode:  apiOpp.ClassificationCode,
		OrganizationType:    apiOpp.OrganizationType,
		ArchiveType:         apiOpp.ArchiveType,
		Active:              apiOpp.Active == "Yes",
	}

	opp.PostedDate = parseAPIDate(apiOpp.PostedDate)
	opp.ResponseDeadline = parseAPIDate(apiOpp.ResponseDeadline)
	opp.ArchiveDate = parseAPIDate(apiOpp.ArchiveDate)

	if apiOpp.OfficeAddress != nil {
		opp.OfficeCity = apiOpp.OfficeAddress.City
		opp.OfficeState = apiOpp.OfficeAddress.State
		opp.OfficeZip = apiOpp.OfficeAddress.ZipCode
		opp.OfficeCountryCode = apiOpp.OfficeAddress.CountryCode
	}

	if apiOpp.PlaceOfPerformance != nil {
		opp.PopStreetAddress = apiOpp.PlaceOfPerformance.StreetAddress
		opp.PopZip = apiOpp.PlaceOfPerformance.Zip
		if apiOpp.PlaceOfPerformance.City != nil {
			opp.PopCity = apiOpp.PlaceOfPerformance.City.Name
		}
		if apiOpp.PlaceOfPerformance.State != nil {
			opp.PopStateCode = apiOpp.PlaceOfPerformance.State.Code
		}
		if apiOpp.PlaceOfPerformance.Country != nil {
			opp.PopCountryCode = apiOpp.PlaceOfPerformance.Country.Code
		}
	}

	for _, contact := range apiOpp.PointOfContact {
		if contact.Type == "primary" {
			opp.PrimaryContactTitle = contact.Title
			opp.PrimaryContactFullname = contact.FullName
			opp.PrimaryContactEmail = contact.Email
			opp.PrimaryContactPhone = contact.Phone
			opp.PrimaryContactFax = contact.Fax
		} else if contact.Type == "secondary" {
			opp.SecondaryContactTitle = contact.Title
			opp.SecondaryContactFullname = contact.FullName
			opp.SecondaryContactEmail = contact.Email
			opp.SecondaryContactPhone = contact.Phone
			opp.SecondaryContactFax = contact.Fax
		}
	}

	return opp
}

func parseAPIDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	formats := []string{
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

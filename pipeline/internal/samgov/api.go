package samgov

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL        = "https://api.sam.gov/opportunities/v2/search"
	defaultLimit   = 1000
	pageDelay      = 1 * time.Second
	requestTimeout = 30 * time.Second
)

// SearchResponse is the top-level SAM.gov opportunities search response.
type SearchResponse struct {
	TotalRecords      int               `json:"totalRecords"`
	Limit             int               `json:"limit"`
	Offset            int               `json:"offset"`
	OpportunitiesData []OpportunityData `json:"opportunitiesData"`
}

type OpportunityData struct {
	NoticeID                  string         `json:"noticeId"`
	Title                     string         `json:"title"`
	SolicitationNumber        string         `json:"solicitationNumber"`
	Type                      string         `json:"type"`
	BaseType                  string         `json:"baseType"`
	PostedDate                string         `json:"postedDate"`
	ResponseDeadLine          string         `json:"responseDeadLine"`
	ArchiveDate               string         `json:"archiveDate"`
	ArchiveType               string         `json:"archiveType"`
	Active                    string         `json:"active"`
	NaicsCode                 string         `json:"naicsCode"`
	ClassificationCode        string         `json:"classificationCode"`
	OrganizationType          string         `json:"organizationType"`
	TypeOfSetAside            *string        `json:"typeOfSetAside"`
	TypeOfSetAsideDescription *string        `json:"typeOfSetAsideDescription"`
	FullParentPathName        string         `json:"fullParentPathName"`
	FullParentPathCode        string         `json:"fullParentPathCode"`
	OfficeAddress             *OfficeAddress `json:"officeAddress"`
	PlaceOfPerformance        *PoP           `json:"placeOfPerformance"`
	PointOfContact            []Contact      `json:"pointOfContact"`
	Award                     *Award         `json:"award"`
	Description               string         `json:"description"`
	UILink                    string         `json:"uiLink"`
	AdditionalInfoLink        *string        `json:"additionalInfoLink"`
	ResourceLinks             []string       `json:"resourceLinks"`
}

type OfficeAddress struct {
	City        string `json:"city"`
	State       string `json:"state"`
	Zipcode     string `json:"zipcode"`
	CountryCode string `json:"countryCode"`
}

type PoP struct {
	StreetAddress string    `json:"streetAddress"`
	City          *NameCode `json:"city"`
	State         *NameCode `json:"state"`
	Country       *NameCode `json:"country"`
	Zip           string    `json:"zip"`
}

type NameCode struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Contact struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Fax      string `json:"fax"`
}

type Award struct {
	Number  string  `json:"number"`
	Date    string  `json:"date"`
	Amount  string  `json:"amount"`
	Awardee Awardee `json:"awardee"`
}

type Awardee struct {
	Name     string          `json:"name"`
	UeiSAM   string          `json:"ueiSAM"`
	Location *AwardeeAddress `json:"location"`
}

type AwardeeAddress struct {
	StreetAddress  string    `json:"streetAddress"`
	StreetAddress2 string    `json:"streetAddress2"`
	City           *NameCode `json:"city"`
	State          *NameCode `json:"state"`
	Country        *NameCode `json:"country"`
	Zip            string    `json:"zip"`
}

// APIRequestLog captures metadata about a single API call for audit logging.
type APIRequestLog struct {
	Endpoint          string
	Method            string
	HTTPStatusCode    *int
	ResponseTimeMs    *int
	ResponseSizeBytes *int
	ErrorMessage      *string
	Success           bool
	APIKeyHash        string
}

// RequestRecorder logs API requests. Implemented by database.DB.
type RequestRecorder interface {
	RecordAPIRequest(ctx context.Context, req *APIRequestLog) error
}

type APIClient struct {
	apiKey     string
	apiKeyHash string
	http       *http.Client
	logger     *slog.Logger
	recorder   RequestRecorder
}

func NewAPIClient(apiKey string, logger *slog.Logger, recorder RequestRecorder) *APIClient {
	hash := sha256.Sum256([]byte(apiKey))
	return &APIClient{
		apiKey:     apiKey,
		apiKeyHash: hex.EncodeToString(hash[:])[:8],
		http: &http.Client{
			Timeout: requestTimeout,
		},
		logger:   logger,
		recorder: recorder,
	}
}

// FetchPage fetches a single page from the SAM.gov opportunities search API.
func (c *APIClient) FetchPage(ctx context.Context, offset, limit int, postedFrom, postedTo string) (*SearchResponse, []json.RawMessage, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", fmt.Sprintf("%d", offset))
	params.Set("postedFrom", postedFrom)
	params.Set("postedTo", postedTo)
	params.Set("ptype", "o,p,k,r,s,g,a,i,u")

	reqURL := baseURL + "?" + params.Encode()

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	elapsed := int(time.Since(start).Milliseconds())

	if err != nil {
		c.logRequest(ctx, elapsed, nil, 0, err)
		return nil, nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logRequest(ctx, elapsed, &resp.StatusCode, 0, err)
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}

	c.logRequest(ctx, elapsed, &resp.StatusCode, len(body), nil)

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	var rawResp struct {
		TotalRecords      int               `json:"totalRecords"`
		Limit             int               `json:"limit"`
		Offset            int               `json:"offset"`
		OpportunitiesData []json.RawMessage `json:"opportunitiesData"`
	}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, nil, fmt.Errorf("unmarshal raw response: %w", err)
	}

	var result SearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, rawResp.OpportunitiesData, nil
}

// FetchAll paginates through all results for a given date window.
func (c *APIClient) FetchAll(ctx context.Context, postedFrom, postedTo string) ([]OpportunityData, []json.RawMessage, int, error) {
	var allOpps []OpportunityData
	var allRaw []json.RawMessage
	offset := 0
	totalRecords := 0
	page := 0

	for {
		page++
		c.logger.Info("fetching API page",
			"page", page,
			"offset", offset,
			"postedFrom", postedFrom,
			"postedTo", postedTo,
		)

		resp, rawItems, err := c.FetchPage(ctx, offset, defaultLimit, postedFrom, postedTo)
		if err != nil {
			return allOpps, allRaw, totalRecords, fmt.Errorf("page %d (offset %d): %w", page, offset, err)
		}

		if page == 1 {
			totalRecords = resp.TotalRecords
			c.logger.Info("total records for window", "total", totalRecords, "postedFrom", postedFrom, "postedTo", postedTo)
		}

		allOpps = append(allOpps, resp.OpportunitiesData...)
		allRaw = append(allRaw, rawItems...)

		if len(resp.OpportunitiesData) < defaultLimit || offset+defaultLimit >= totalRecords {
			break
		}

		offset += defaultLimit

		select {
		case <-ctx.Done():
			return allOpps, allRaw, totalRecords, ctx.Err()
		case <-time.After(pageDelay):
		}
	}

	return allOpps, allRaw, totalRecords, nil
}

func (c *APIClient) logRequest(ctx context.Context, elapsedMs int, statusCode *int, responseSize int, reqErr error) {
	if c.recorder == nil {
		return
	}

	req := &APIRequestLog{
		Endpoint:       "opportunities/v2/search",
		Method:         "GET",
		ResponseTimeMs: &elapsedMs,
		Success:        reqErr == nil && statusCode != nil && *statusCode == http.StatusOK,
		APIKeyHash:     c.apiKeyHash,
	}
	if statusCode != nil {
		req.HTTPStatusCode = statusCode
	}
	if responseSize > 0 {
		req.ResponseSizeBytes = &responseSize
	}
	if reqErr != nil {
		errMsg := reqErr.Error()
		req.ErrorMessage = &errMsg
	}

	if err := c.recorder.RecordAPIRequest(ctx, req); err != nil {
		c.logger.Warn("failed to log SAM.gov API request", "error", err)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

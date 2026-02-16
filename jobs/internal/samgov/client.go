package samgov

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/handriss/govtrove/jobs/internal/database"
)

// TrackedHTTPClient wraps HTTP requests to SAM.gov and logs all requests
type TrackedHTTPClient struct {
	httpClient     *http.Client
	db             *database.DB
	logger         *slog.Logger
	ingestionRunID *int
}

func NewTrackedHTTPClient(db *database.DB, logger *slog.Logger) *TrackedHTTPClient {
	return &TrackedHTTPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
		db:     db,
		logger: logger,
	}
}

func (c *TrackedHTTPClient) SetIngestionRunID(runID int) {
	c.ingestionRunID = &runID
}

// TrackedResponse wraps http.Response with additional tracking info
type TrackedResponse struct {
	*http.Response
	ResponseTimeMs int
}

// Do performs an HTTP request and records usage to the database
func (c *TrackedHTTPClient) Do(ctx context.Context, req *http.Request) (*TrackedResponse, error) {
	startTime := time.Now()

	var requestParams json.RawMessage
	if req.URL.RawQuery != "" {
		params := make(map[string]string)
		for key, values := range req.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		requestParams, _ = json.Marshal(params)
	}

	resp, err := c.httpClient.Do(req)
	responseTimeMs := int(time.Since(startTime).Milliseconds())

	samgovReq := &database.SAMGovRequest{
		Endpoint:       c.sanitizeEndpoint(req.URL),
		Method:         req.Method,
		ResponseTimeMs: &responseTimeMs,
		RequestParams:  requestParams,
		IngestionRunID: c.ingestionRunID,
	}

	if err != nil {
		errMsg := err.Error()
		samgovReq.ErrorMessage = &errMsg
		samgovReq.Success = false
	} else {
		samgovReq.HTTPStatusCode = &resp.StatusCode
		samgovReq.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	if c.db != nil {
		if _, dbErr := c.db.RecordSAMGovRequest(ctx, samgovReq); dbErr != nil {
			c.logger.Warn("failed to record SAM.gov request", "error", dbErr)
		}
	}

	c.logger.Debug("SAM.gov request",
		"endpoint", samgovReq.Endpoint,
		"method", req.Method,
		"status_code", samgovReq.HTTPStatusCode,
		"response_time_ms", responseTimeMs,
		"success", samgovReq.Success,
	)

	if err != nil {
		return nil, err
	}

	return &TrackedResponse{
		Response:       resp,
		ResponseTimeMs: responseTimeMs,
	}, nil
}

// Get performs a GET request and records usage
func (c *TrackedHTTPClient) Get(ctx context.Context, url string) (*TrackedResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return c.Do(ctx, req)
}

// GetWithAPIKey performs a GET request with the API key in the header
func (c *TrackedHTTPClient) GetWithAPIKey(ctx context.Context, url string, apiKey string) (*TrackedResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", apiKey)
	return c.Do(ctx, req)
}

// sanitizeEndpoint removes query params from URL for cleaner logging
func (c *TrackedHTTPClient) sanitizeEndpoint(u *url.URL) string {
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}

// StreamingResponse wraps a response body and tracks bytes read
type StreamingResponse struct {
	body           io.ReadCloser
	bytesRead      int
	db             *database.DB
	ctx            context.Context
	usageID        int
	responseTimeMs int
}

func (s *StreamingResponse) Read(p []byte) (n int, err error) {
	n, err = s.body.Read(p)
	s.bytesRead += n
	return n, err
}

func (s *StreamingResponse) Close() error {
	if s.db != nil && s.usageID > 0 {
		s.db.UpdateSAMGovRequestResponseSize(s.ctx, s.usageID, s.bytesRead)
	}
	return s.body.Close()
}

func (s *StreamingResponse) BytesRead() int {
	return s.bytesRead
}

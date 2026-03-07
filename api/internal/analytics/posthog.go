package analytics

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

var (
	apiKey string
	host   string
	logger *slog.Logger
)

func Init(key, h string, l *slog.Logger) {
	apiKey = key
	host = h
	logger = l
}

type capturePayload struct {
	APIKey     string         `json:"api_key"`
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Properties map[string]any `json:"properties,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

func CaptureEvent(distinctID, event string, properties map[string]any) {
	if apiKey == "" || distinctID == "" {
		return
	}

	payload := capturePayload{
		APIKey:     apiKey,
		Event:      event,
		DistinctID: distinctID,
		Properties: properties,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			if logger != nil {
				logger.Debug("posthog: marshal error", "error", err)
			}
			return
		}

		endpoint := host + "/i/v0/e/"
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
		if err != nil {
			if logger != nil {
				logger.Debug("posthog: request error", "error", err)
			}
			return
		}
		resp.Body.Close()
	}()
}

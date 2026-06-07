package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatusHandler struct {
	pool *pgxpool.Pool
}

func NewStatusHandler(pool *pgxpool.Pool) *StatusHandler {
	return &StatusHandler{pool: pool}
}

type StatusResponse struct {
	LastSyncedAt          *time.Time `json:"last_synced_at"`
	DurationSeconds       *int       `json:"duration_seconds,omitempty"`
	OpportunitiesUpdated  *int       `json:"opportunities_updated,omitempty"`
	OpportunitiesUnchanged *int      `json:"opportunities_unchanged,omitempty"`
	TotalOpportunities    *int       `json:"total_opportunities,omitempty"`
	Source                string     `json:"source"`
}

func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := StatusResponse{Source: "SAM.gov"}

	var completedAt *time.Time
	var durationMs *int
	var statsJSON *string
	err := h.pool.QueryRow(ctx,
		`SELECT completed_at, duration_ms, stats::text
		 FROM pipeline.pipeline_steps
		 WHERE step_name = 'reconcile' AND status = 'completed'
		 ORDER BY completed_at DESC LIMIT 1`,
	).Scan(&completedAt, &durationMs, &statsJSON)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.LastSyncedAt = completedAt

	if durationMs != nil {
		secs := int(math.Round(float64(*durationMs) / 1000))
		resp.DurationSeconds = &secs
	}

	if statsJSON != nil {
		var stats map[string]any
		if json.Unmarshal([]byte(*statsJSON), &stats) == nil {
			if v, ok := stats["upserted"].(float64); ok {
				i := int(v)
				resp.OpportunitiesUpdated = &i
			}
			if v, ok := stats["touched"].(float64); ok {
				i := int(v)
				resp.OpportunitiesUnchanged = &i
			}
			if resp.OpportunitiesUpdated != nil && resp.OpportunitiesUnchanged != nil {
				total := *resp.OpportunitiesUpdated + *resp.OpportunitiesUnchanged
				resp.TotalOpportunities = &total
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

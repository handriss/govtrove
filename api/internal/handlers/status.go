package handlers

import (
	"context"
	"encoding/json"
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
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var resp StatusResponse
	err := h.pool.QueryRow(ctx,
		`SELECT completed_at FROM ingestion_runs
		 WHERE status = 'completed'
		 ORDER BY completed_at DESC LIMIT 1`,
	).Scan(&resp.LastSyncedAt)

	if err != nil {
		resp.LastSyncedAt = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

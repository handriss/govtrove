package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

// Check is a liveness probe only — it intentionally does NOT touch the database.
// App Runner hits this every 10s; pinging the DB here kept the Neon compute
// awake 24/7 (it never reached the idle threshold to scale to zero). Use
// /health/db (CheckDB) for an on-demand DB-reachability check.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

// CheckDB verifies database reachability (the pre-2026-08 /health behavior).
// It is deliberately NOT wired to any external/automated health check, so
// routine liveness probes don't hold the Neon compute awake. Call it manually.
func (h *HealthHandler) CheckDB(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "error"
	}

	status := "ok"
	if dbStatus != "ok" {
		status = "degraded"
	}

	resp := HealthResponse{
		Status:   status,
		Database: dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}

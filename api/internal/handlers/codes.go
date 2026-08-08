package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type CodeHandler struct {
	repo           *repository.CodeRepository
	mcpInternalURL string
	internalToken  string
	logger         *slog.Logger
	httpClient     *http.Client
}

func NewCodeHandler(repo *repository.CodeRepository, mcpInternalURL, internalToken string, logger *slog.Logger) *CodeHandler {
	return &CodeHandler{
		repo:           repo,
		mcpInternalURL: mcpInternalURL,
		internalToken:  internalToken,
		logger:         logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (h *CodeHandler) MatchNAICS(w http.ResponseWriter, r *http.Request) {
	h.matchCodes(w, r, "naics")
}

func (h *CodeHandler) MatchPSC(w http.ResponseWriter, r *http.Request) {
	h.matchCodes(w, r, "psc")
}

func (h *CodeHandler) matchCodes(w http.ResponseWriter, r *http.Request, codeType string) {
	var req models.CodeMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}
	if len(req.Description) > 2000 {
		http.Error(w, "description must be 2000 characters or less", http.StatusBadRequest)
		return
	}
	if req.Limit <= 0 || req.Limit > 25 {
		req.Limit = 10
	}

	embedding, err := h.getEmbedding(req.Description)
	if err != nil {
		h.logger.Error("failed to get embedding", "error", err)
		http.Error(w, "Failed to process description", http.StatusInternalServerError)
		return
	}

	matches, err := h.repo.FindSimilar(r.Context(), codeType, embedding, req.Limit)
	if err != nil {
		h.logger.Error("failed to find similar codes", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	matchedCodes := make([]string, len(matches))
	for i, m := range matches {
		matchedCodes[i] = m.Code
	}

	// Active-opportunity count per matched code (same predicate as an app search for
	// that code), so users can see which codes actually have opportunities before clicking.
	if counts, err := h.repo.ActiveCountsByCode(r.Context(), codeType, matchedCodes); err != nil {
		h.logger.Error("failed to count active opportunities", "error", err)
	} else {
		for i := range matches {
			matches[i].ActiveCount = counts[matches[i].Code]
		}
	}

	correlations, err := h.repo.GetCorrelations(r.Context(), codeType, matchedCodes, 10)
	if err != nil {
		h.logger.Error("failed to get correlations", "error", err)
		correlations = nil
	}

	if matches == nil {
		matches = []models.CodeMatch{}
	}
	if correlations == nil {
		correlations = []models.CodeCorrelation{}
	}

	// Fire-and-forget usage logging — never block or fail the response on it.
	go func(desc string, ms []models.CodeMatch) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var topCode string
		var topScore float64
		if len(ms) > 0 {
			topCode = ms[0].Code
			topScore = ms[0].Similarity
		}
		if err := h.repo.LogLookup(ctx, codeType, desc, len(ms), topCode, topScore); err != nil {
			h.logger.Warn("failed to log code lookup", "error", err)
		}
	}(req.Description, matches)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CodeMatchResponse{
		Matches:      matches,
		Correlations: correlations,
	})
}

func (h *CodeHandler) GetCorrelations(w http.ResponseWriter, r *http.Request) {
	codeType := r.URL.Query().Get("type")
	if codeType != "naics" && codeType != "psc" {
		http.Error(w, "type must be 'naics' or 'psc'", http.StatusBadRequest)
		return
	}

	codesParam := r.URL.Query().Get("codes")
	if codesParam == "" {
		http.Error(w, "codes parameter is required", http.StatusBadRequest)
		return
	}
	codes := strings.Split(codesParam, ",")
	if len(codes) > 25 {
		codes = codes[:25]
	}

	correlations, err := h.repo.GetCorrelations(r.Context(), codeType, codes, 20)
	if err != nil {
		h.logger.Error("failed to get correlations", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if correlations == nil {
		correlations = []models.CodeCorrelation{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"correlations": correlations})
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (h *CodeHandler) getEmbedding(text string) ([]float32, error) {
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, fmt.Errorf("marshaling embed request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.mcpInternalURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", h.internalToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling embed endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding embed response: %w", err)
	}

	if len(result.Embedding) != 384 {
		return nil, fmt.Errorf("unexpected embedding dimension: %d", len(result.Embedding))
	}

	return result.Embedding, nil
}

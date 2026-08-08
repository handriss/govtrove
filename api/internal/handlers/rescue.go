package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

// RescueSearch diagnoses a zero-result search and returns verified
// alternative searches. The client calls it after a search comes back
// empty; the empty state upgrades progressively when this responds.
func (h *OpportunityHandler) RescueSearch(w http.ResponseWriter, r *http.Request) {
	if h.rescue == nil {
		http.NotFound(w, r)
		return
	}

	params := h.parseSearchParams(r)

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	spellFix := ""
	if q := strings.TrimSpace(strings.ReplaceAll(params.Query, `"`, "")); q != "" {
		if suggestion, err := h.repo.SuggestQuery(ctx, q); err == nil {
			spellFix = suggestion
		}
	}

	result := h.rescue.Rescue(ctx, params, 0, spellFix)
	if result == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.writeJSON(w, http.StatusOK, result)

	userID := h.resolveOptionalUserID(r)
	event := &models.SearchEvent{
		EventType: "rescue_shown",
		Filters: map[string]interface{}{
			"stage":       result.Stage,
			"cause":       result.Cause,
			"suggestions": len(result.Suggestions),
		},
	}
	if params.Query != "" {
		event.Query = &params.Query
	}
	n := len(result.Suggestions)
	event.TotalResults = &n
	h.eventLog.Log(r, userID, event)
}

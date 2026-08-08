package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRescueSearchDisabledReturns404(t *testing.T) {
	h := &OpportunityHandler{}
	w := httptest.NewRecorder()
	h.RescueSearch(w, httptest.NewRequest(http.MethodGet, "/api/opportunities/rescue?q=test", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when rescue is not enabled", w.Code)
	}
}

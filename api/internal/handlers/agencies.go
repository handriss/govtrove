package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/handriss/govtrove/api/internal/repository"
)

type AgencyHandler struct {
	repo   *repository.AgencyRepository
	logger *slog.Logger
}

func NewAgencyHandler(repo *repository.AgencyRepository, logger *slog.Logger) *AgencyHandler {
	return &AgencyHandler{repo: repo, logger: logger}
}

var breadcrumbAbbreviations = map[string]string{
	"DEPT OF DEFENSE":                                  "DoD",
	"DEPT OF THE ARMY":                                 "Army",
	"DEPT OF THE NAVY":                                 "Navy",
	"DEPT OF THE AIR FORCE":                            "Air Force",
	"GENERAL SERVICES ADMINISTRATION":                  "GSA",
	"VETERANS AFFAIRS, DEPARTMENT OF":                   "VA",
	"HOMELAND SECURITY, DEPARTMENT OF":                  "DHS",
	"HEALTH AND HUMAN SERVICES, DEPARTMENT OF":          "HHS",
	"NATIONAL AERONAUTICS AND SPACE ADMINISTRATION":     "NASA",
	"AGRICULTURE, DEPARTMENT OF":                        "USDA",
	"COMMERCE, DEPARTMENT OF":                           "Commerce",
	"ENERGY, DEPARTMENT OF":                             "DOE",
	"INTERIOR, DEPARTMENT OF THE":                       "Interior",
	"JUSTICE, DEPARTMENT OF":                            "DOJ",
	"LABOR, DEPARTMENT OF":                              "DOL",
	"STATE, DEPARTMENT OF":                              "State",
	"TRANSPORTATION, DEPARTMENT OF":                     "DOT",
	"TREASURY, DEPARTMENT OF THE":                       "Treasury",
	"ENVIRONMENTAL PROTECTION AGENCY":                   "EPA",
	"SMALL BUSINESS ADMINISTRATION":                     "SBA",
	"SOCIAL SECURITY ADMINISTRATION":                    "SSA",
	"OFFICE OF PERSONNEL MANAGEMENT":                    "OPM",
	"EDUCATION, DEPARTMENT OF":                          "Education",
	"HOUSING AND URBAN DEVELOPMENT, DEPARTMENT OF":      "HUD",
}

func makeBreadcrumb(parentPath string) string {
	parts := strings.Split(parentPath, ".")
	abbreviated := make([]string, len(parts))
	for i, part := range parts {
		if short, ok := breadcrumbAbbreviations[part]; ok {
			abbreviated[i] = short
		} else if len(part) > 30 {
			abbreviated[i] = part[:30] + "..."
		} else {
			abbreviated[i] = part
		}
	}
	return strings.Join(abbreviated, " > ")
}

type agencyResponse struct {
	Name             string  `json:"name"`
	ShortName        *string `json:"short_name,omitempty"`
	Level            string  `json:"level"`
	ParentPath       string  `json:"parent_path"`
	Breadcrumb       string  `json:"breadcrumb"`
	OpportunityCount int     `json:"count"`
}

func (h *AgencyHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		q = q[:200]
	}

	limit := 15
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	agencies, err := h.repo.Search(r.Context(), q, limit)
	if err != nil {
		h.logger.Error("agency search failed", "error", err, "query", q)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	results := make([]agencyResponse, len(agencies))
	for i, a := range agencies {
		results[i] = agencyResponse{
			Name:             a.Name,
			ShortName:        a.ShortName,
			Level:            a.Level,
			ParentPath:       a.ParentPath,
			Breadcrumb:       makeBreadcrumb(a.ParentPath),
			OpportunityCount: a.OpportunityCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(map[string]any{
		"agencies": results,
	})
}

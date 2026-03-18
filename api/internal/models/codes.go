package models

type CodeMatchRequest struct {
	Description string `json:"description"`
	Limit       int    `json:"limit"`
}

type CodeMatch struct {
	Code       string  `json:"code"`
	Title      string  `json:"title"`
	Level      int     `json:"level"`
	Similarity float64 `json:"similarity"`
}

type CodeCorrelation struct {
	Code            string `json:"code"`
	Title           string `json:"title"`
	CoOccurrenceCount int  `json:"co_occurrence_count"`
}

type CodeMatchResponse struct {
	Matches      []CodeMatch       `json:"matches"`
	Correlations []CodeCorrelation `json:"correlations"`
}

type CorrelationRequest struct {
	CodeType string   `json:"type"`
	Codes    []string `json:"codes"`
}

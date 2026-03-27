package models

type GeoSynonym struct {
	ID        int      `json:"id"`
	Term      string   `json:"term"`
	MatchType string   `json:"match_type"`
	States    []string `json:"states"`
	Cities    []string `json:"cities"`
}

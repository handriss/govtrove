package models

type SearchAnalytics struct {
	PopularSearches []SearchTermStat `json:"popular_searches"`
	ZeroResultTerms []SearchTermStat `json:"zero_result_searches"`
	FilterUsage     FilterUsageStats `json:"filter_usage"`
	ClickStats      ClickStats       `json:"click_stats"`
	EventCounts     EventCounts      `json:"event_counts"`
}

type SearchTermStat struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

type FilterUsageStats struct {
	Types     []FilterStat `json:"types"`
	SetAsides []FilterStat `json:"set_asides"`
	States    []FilterStat `json:"states"`
}

type FilterStat struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type ClickStats struct {
	TotalClicks      int            `json:"total_clicks"`
	TotalSearches    int            `json:"total_searches"`
	ClickThroughRate float64        `json:"click_through_rate"`
	ByPosition       []PositionStat `json:"by_position"`
}

type PositionStat struct {
	Position int `json:"position"`
	Clicks   int `json:"clicks"`
}

type EventCounts struct {
	Searches int `json:"searches"`
	Filters  int `json:"filters"`
	Clicks   int `json:"clicks"`
	Pages    int `json:"pages"`
}

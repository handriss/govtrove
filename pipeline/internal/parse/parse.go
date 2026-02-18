package parse

import (
	"strconv"
	"strings"
	"time"
)

func Date(s string) *time.Time {
	if s == "" {
		return nil
	}

	formats := []string{
		"2006-01-02 15:04:05.000-07",
		"2006-01-02 15:04:05.00-07",
		"2006-01-02 15:04:05.0-07",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05.000-0700",
		"2006-01-02 15:04:05-0700",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006 15:04",
		"01/02/2006",
		"0102/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

func Active(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "yes" || s == "true" || s == "1"
}

// DateOnly parses a date string and truncates to midnight UTC.
// Returns nil for empty or unparseable values.
func DateOnly(s string) *time.Time {
	t := Date(s)
	if t == nil {
		return nil
	}
	truncated := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &truncated
}

func IsSentinelDate(t *time.Time) bool {
	if t == nil {
		return false
	}
	return t.Year() < 1980
}

func Amount(s string) *float64 {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.TrimSpace(s)

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return &f
	}
	return nil
}

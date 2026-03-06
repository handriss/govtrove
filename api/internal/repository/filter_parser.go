package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

type savedFilters struct {
	Keyword        string   `json:"keyword"`
	NAICS          []string `json:"naics"`
	PSC            []string `json:"psc"`
	SetAside       []string `json:"setAside"`
	Department     string   `json:"department"`
	State          string   `json:"state"`
	NoticeType     []string `json:"noticeType"`
	DeadlinePreset string   `json:"deadlinePreset"`
	PostedFrom     string   `json:"postedFrom"`
	PostedTo       string   `json:"postedTo"`
	DeadlineFrom   string   `json:"deadlineFrom"`
	DeadlineTo     string   `json:"deadlineTo"`
}

func ParseSavedFilters(raw json.RawMessage) (models.SearchParams, error) {
	var f savedFilters
	if err := json.Unmarshal(raw, &f); err != nil {
		return models.SearchParams{}, fmt.Errorf("unmarshal filters: %w", err)
	}

	p := models.SearchParams{
		Query:      f.Keyword,
		Types:      models.NormalizeNoticeTypes(f.NoticeType),
		SetAsides:  f.SetAside,
		NAICSCodes: f.NAICS,
		PSCCodes:   f.PSC,
		Department: f.Department,
		Sort:       "posted_date",
		Order:      "desc",
	}

	if f.State != "" {
		p.States = []string{f.State}
	}

	if f.PostedFrom != "" {
		if t, err := time.Parse("2006-01-02", f.PostedFrom); err == nil {
			p.PostedFrom = &t
		}
	}
	if f.PostedTo != "" {
		if t, err := time.Parse("2006-01-02", f.PostedTo); err == nil {
			p.PostedTo = &t
		}
	}

	if f.DeadlinePreset != "" {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		p.DeadlineFrom = &today
		to := deadlinePresetToDate(f.DeadlinePreset, now)
		if to != nil {
			p.DeadlineTo = to
		}
	} else {
		if f.DeadlineFrom != "" {
			if t, err := time.Parse("2006-01-02", f.DeadlineFrom); err == nil {
				p.DeadlineFrom = &t
			}
		}
		if f.DeadlineTo != "" {
			if t, err := time.Parse("2006-01-02", f.DeadlineTo); err == nil {
				p.DeadlineTo = &t
			}
		}
	}

	return p, nil
}

func deadlinePresetToDate(preset string, now time.Time) *time.Time {
	switch preset {
	case "7":
		t := now.AddDate(0, 0, 7)
		return &t
	case "14":
		t := now.AddDate(0, 0, 14)
		return &t
	case "30":
		t := now.AddDate(0, 0, 30)
		return &t
	case "60":
		t := now.AddDate(0, 0, 60)
		return &t
	case "90":
		t := now.AddDate(0, 0, 90)
		return &t
	case "quarter":
		endMonth := ((int(now.Month())-1)/3+1)*3 + 1
		year := now.Year()
		if endMonth > 12 {
			endMonth -= 12
			year++
		}
		t := time.Date(year, time.Month(endMonth), 0, 0, 0, 0, 0, time.UTC)
		return &t
	}
	return nil
}

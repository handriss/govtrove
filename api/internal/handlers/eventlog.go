package handlers

import (
	"log/slog"
	"net/http"

	"github.com/handriss/govtrove/api/internal/botfilter"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

type EventLogger struct {
	repo   *repository.EventRepository
	logger *slog.Logger
}

func NewEventLogger(repo *repository.EventRepository, logger *slog.Logger) *EventLogger {
	return &EventLogger{repo: repo, logger: logger}
}

func (l *EventLogger) Log(r *http.Request, userID *int, event *models.SearchEvent) {
	event.UserID = userID

	ua := r.Header.Get("User-Agent")
	if ua != "" {
		if len(ua) > 512 {
			ua = ua[:512]
		}
		event.UserAgent = &ua
	}

	ref := r.Header.Get("Referer")
	if ref != "" {
		if len(ref) > 2048 {
			ref = ref[:2048]
		}
		event.Referer = &ref
	}

	if botfilter.IsBot(ua) {
		return
	}

	if err := l.repo.Create(r.Context(), event); err != nil {
		l.logger.Error("failed to log event", "event_type", event.EventType, "error", err)
	}
}

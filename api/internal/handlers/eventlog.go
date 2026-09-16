package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/handriss/govtrove/api/internal/botfilter"
	"github.com/handriss/govtrove/api/internal/models"
	"github.com/handriss/govtrove/api/internal/repository"
)

// eventWriteTimeout bounds the detached analytics insert: the caller is a live
// request handler, so this must not become an unbounded wait.
const eventWriteTimeout = 5 * time.Second

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

	utmCampaign := r.Header.Get("X-UTM-Campaign")
	if utmCampaign != "" {
		if len(utmCampaign) > 255 {
			utmCampaign = utmCampaign[:255]
		}
		event.UtmCampaign = &utmCampaign
	}

	if botfilter.IsBot(ua) {
		return
	}

	// Analytics outlive the request that produced them. Inheriting the request
	// context cancelled the insert whenever the visitor navigated away mid-flight,
	// which silently dropped the event and under-counted every search figure.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), eventWriteTimeout)
	defer cancel()

	if err := l.repo.Create(ctx, event); err != nil {
		l.logger.Error("failed to log event", "event_type", event.EventType, "error", err)
	}
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/getsentry/sentry-go"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/config"
	"github.com/handriss/govtrove/api/internal/email"
	"github.com/handriss/govtrove/api/internal/handlers"
	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/ogimage"
	"github.com/handriss/govtrove/api/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      "production",
			AttachStacktrace: true,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "sentry init failed: %v\n", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Log that we're starting (helps debug container startup issues)
	logger.Info("starting database connection", "url_length", len(cfg.DatabaseURL))

	// Retry database connection with exponential backoff
	var pool *pgxpool.Pool
	for attempt := 1; attempt <= 5; attempt++ {
		connCtx, connCancel := context.WithTimeout(ctx, 30*time.Second)
		pool, err = pgxpool.New(connCtx, cfg.DatabaseURL)
		connCancel()
		if err != nil {
			logger.Error("failed to create pool", "error", err, "attempt", attempt)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
		err = pool.Ping(pingCtx)
		pingCancel()
		if err != nil {
			logger.Error("failed to ping database", "error", err, "attempt", attempt)
			pool.Close()
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}
		break
	}
	if err != nil {
		logger.Error("all database connection attempts failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	var snsClient *sns.Client
	if cfg.SNSTopicARN != "" {
		awsCfg, awsErr := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
		if awsErr != nil {
			logger.Warn("failed to load AWS config, SNS disabled", "error", awsErr)
		} else {
			snsClient = sns.NewFromConfig(awsCfg)
			logger.Info("SNS client configured", "topic_arn", cfg.SNSTopicARN)
		}
	}

	var emailSvc *email.Service
	if cfg.ResendAPIKey != "" {
		fromEmail := cfg.ResendFromEmail
		if fromEmail == "" {
			fromEmail = "notifications@govtrove.com"
		}
		emailSvc = email.NewService(cfg.ResendAPIKey, fromEmail, cfg.ResendWebhookSecret, pool, logger)
		logger.Info("Resend email service configured", "from", fromEmail)
	}

	// JWKS for WorkOS JWT validation
	var jwks keyfunc.Keyfunc
	if cfg.WorkOSClientID != "" {
		jwksURL := fmt.Sprintf("https://api.workos.com/sso/jwks/%s", cfg.WorkOSClientID)
		k, jwksErr := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
		if jwksErr != nil {
			logger.Error("failed to initialize JWKS", "error", jwksErr)
			os.Exit(1)
		}
		jwks = k
		authmw.SetClientID(cfg.WorkOSClientID)
		logger.Info("JWKS initialized", "client_id", cfg.WorkOSClientID)
	} else {
		logger.Warn("WORKOS_CLIENT_ID not set, auth endpoints disabled")
	}

	ogRenderer, err := ogimage.NewRenderer()
	if err != nil {
		logger.Error("failed to initialize OG image renderer", "error", err)
		os.Exit(1)
	}
	logger.Info("OG image renderer initialized")

	oppRepo := repository.NewOpportunityRepository(pool)
	agencyRepo := repository.NewAgencyRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	contactRepo := repository.NewContactRepository(pool)
	accountRequestRepo := repository.NewAccountRequestRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	savedOppRepo := repository.NewSavedOpportunityRepository(pool)
	savedSearchRepo := repository.NewSavedSearchRepository(pool)
	userUpdateRepo := repository.NewUserUpdateRepository(pool)
	pipelineRepo := repository.NewPipelineRepository(pool)
	utmRepo := repository.NewUTMRepository(pool)
	emailPrefsRepo := repository.NewEmailPreferencesRepository(pool)
	sentEmailsRepo := repository.NewSentEmailsRepository(pool)

	eventLog := handlers.NewEventLogger(eventRepo, logger)
	oppHandler := handlers.NewOpportunityHandler(oppRepo, ogRenderer, logger, eventLog, userRepo)
	agencyHandler := handlers.NewAgencyHandler(agencyRepo, logger)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo, logger)
	contactHandler := handlers.NewContactHandler(contactRepo, snsClient, cfg.SNSTopicARN, logger)
	accountRequestHandler := handlers.NewAccountRequestHandler(accountRequestRepo, userRepo, snsClient, cfg.SNSTopicARN, logger)
	adminHandler := handlers.NewAdminHandler(userRepo, userUpdateRepo, pipelineRepo, emailPrefsRepo, sentEmailsRepo, emailSvc, logger)
	userHandler := handlers.NewUserHandler(userRepo, logger)
	authHandler := handlers.NewAuthHandler(userRepo, emailPrefsRepo, emailSvc, logger)
	savedOppHandler := handlers.NewSavedOpportunityHandler(savedOppRepo, userRepo, logger, eventLog)
	savedSearchHandler := handlers.NewSavedSearchHandler(savedSearchRepo, userRepo, logger, eventLog)
	userUpdateHandler := handlers.NewUserUpdateHandler(userUpdateRepo, userRepo, logger)
	utmHandler := handlers.NewUTMHandler(utmRepo, logger)
	webhookHandler := handlers.NewWebhookHandler(sentEmailsRepo, emailPrefsRepo, emailSvc, logger)
	unsubscribeHandler := handlers.NewUnsubscribeHandler(emailPrefsRepo, emailSvc, logger)
	preferencesHandler := handlers.NewPreferencesHandler(emailPrefsRepo, userRepo, logger)
	healthHandler := handlers.NewHealthHandler(pool)
	statusHandler := handlers.NewStatusHandler(pool)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(authmw.SentryMiddleware)
	r.Use(middleware.Logger)
	r.Use(authmw.SentryRecoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			next.ServeHTTP(w, r)
		})
	})

	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-UTM-Campaign"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler.Check)
	r.Get("/og/opportunities/{id}/card.png", oppHandler.GetOGImage)
	r.Get("/og/opportunities/{id}", oppHandler.GetOGCard)
	// Admin access: check is_admin column first, fall back to ADMIN_EMAILS allowlist
	adminEmailSet := make(map[string]bool)
	if cfg.AdminEmails != "" {
		for _, e := range strings.Split(cfg.AdminEmails, ",") {
			if trimmed := strings.TrimSpace(e); trimmed != "" {
				adminEmailSet[strings.ToLower(trimmed)] = true
			}
		}
	}
	adminLookup := func(ctx context.Context, workosID string) (bool, error) {
		isAdmin, err := userRepo.IsAdmin(ctx, workosID)
		if err == nil && isAdmin {
			return true, nil
		}
		if len(adminEmailSet) > 0 {
			user, userErr := userRepo.GetByWorkOSID(ctx, workosID)
			if userErr == nil && user != nil && adminEmailSet[strings.ToLower(user.Email)] {
				return true, nil
			}
		}
		return false, err
	}

	r.Route("/api", func(r chi.Router) {
		r.Use(httprate.LimitByIP(100, time.Minute))
		r.Use(authmw.MaxBodySize(1 << 20))

		r.Post("/webhooks/resend", webhookHandler.HandleResend)
		r.Get("/unsubscribe", unsubscribeHandler.HandleUnsubscribe)

		r.Group(func(r chi.Router) {
			if jwks != nil {
				r.Use(authmw.OptionalAuth(jwks))
			}
			r.Get("/opportunities", oppHandler.Search)
			r.Get("/opportunities/{id}", oppHandler.GetByID)
		})

		r.Route("/utm", func(r chi.Router) {
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/", utmHandler.TrackVisit)
		})

		r.Get("/agencies", agencyHandler.Search)
		r.Get("/opportunities/facets", oppHandler.GetFacets)
		r.Get("/opportunities/{id}/history", oppHandler.GetSolicitationHistory)
		r.Get("/filters", oppHandler.GetFilters)
		r.Get("/status", statusHandler.GetStatus)
		r.Route("/contact", func(r chi.Router) {
			r.Use(httprate.LimitByIP(5, time.Hour))
			r.Post("/", contactHandler.Create)
		})

		if jwks != nil {
			r.Group(func(r chi.Router) {
				r.Use(authmw.RequireAuth(jwks))
				r.Get("/me", userHandler.GetMe)
				r.Post("/auth/sync", authHandler.Sync)
				r.Post("/account/requests", accountRequestHandler.Create)
				r.Get("/preferences", preferencesHandler.Get)
				r.Put("/preferences", preferencesHandler.Update)

				r.Route("/admin", func(r chi.Router) {
					r.Use(authmw.RequireAdmin(adminLookup))
					r.Get("/analytics", analyticsHandler.GetAnalytics)
					r.Get("/users", adminHandler.ListUsers)
					r.Get("/notifications", adminHandler.ListNotifications)
					r.Get("/api-keys", adminHandler.ListApiKeys)
					r.Get("/samgov-requests", adminHandler.ListSamgovRequests)
					r.Get("/api-key-usage", adminHandler.GetApiKeyUsage)
					r.Get("/pipeline-runs", adminHandler.ListPipelineRuns)
					r.Get("/pipeline-runs/{id}", adminHandler.GetPipelineRunDetail)
					r.Get("/search-events", adminHandler.ListSearchEvents)
					r.Get("/data-quality", adminHandler.ListDataQualityIssues)
					r.Get("/data-quality/{id}", adminHandler.GetDataQualityDetail)
					r.Put("/data-quality/{id}", adminHandler.UpdateDataQualityResolution)
					r.Get("/reconcile-dq", adminHandler.ListReconcileDQIssues)
					r.Get("/reconcile-dq/{id}", adminHandler.GetReconcileDQDetail)
					r.Put("/reconcile-dq/{id}", adminHandler.UpdateReconcileDQResolution)
					r.Get("/utm-analytics", utmHandler.GetAnalytics)
					r.Get("/email-preferences", adminHandler.ListEmailPreferences)
					r.Put("/email-preferences/{userId}", adminHandler.UpdateEmailPreference)
					r.Get("/sent-emails", adminHandler.ListSentEmails)
					r.Post("/sent-emails/{id}/resend", adminHandler.ResendEmail)
					r.Post("/send-email", adminHandler.SendNewEmail)
				})

				r.Get("/saved/opportunities", savedOppHandler.ListWithDetails)
				r.Get("/saved/opportunities/ids", savedOppHandler.ListIDs)
				r.Post("/saved/opportunities", savedOppHandler.Add)
				r.Post("/saved/opportunities/bulk", savedOppHandler.BulkAdd)
				r.Put("/saved/opportunities/{id}", savedOppHandler.UpdateNotes)
				r.Delete("/saved/opportunities", savedOppHandler.Remove)
				r.Delete("/saved/opportunities/by-opportunity/{opportunityId}", savedOppHandler.RemoveByOpportunityID)

				r.Get("/saved/searches", savedSearchHandler.List)
				r.Post("/saved/searches", savedSearchHandler.Create)
				r.Put("/saved/searches/{id}", savedSearchHandler.Update)
				r.Delete("/saved/searches/{id}", savedSearchHandler.Delete)
				r.Post("/saved/searches/{id}/run", savedSearchHandler.Run)

				r.Get("/updates", userUpdateHandler.List)
				r.Get("/updates/count", userUpdateHandler.Count)
				r.Put("/updates/{id}/read", userUpdateHandler.MarkRead)
				r.Put("/updates/read-all", userUpdateHandler.MarkAllRead)
				r.Delete("/updates/{id}", userUpdateHandler.Delete)
			})
		}
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server")
		sentry.Flush(2 * time.Second)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		cancel()
	}()

	logger.Info("starting server", "port", cfg.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

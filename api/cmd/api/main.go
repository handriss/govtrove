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
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handriss/govtrove/api/internal/config"
	"github.com/handriss/govtrove/api/internal/handlers"
	authmw "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
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
			logger.Warn("failed to load AWS config, SNS notifications disabled", "error", awsErr)
		} else {
			snsClient = sns.NewFromConfig(awsCfg)
			logger.Info("SNS client configured", "topic_arn", cfg.SNSTopicARN)
		}
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

	oppRepo := repository.NewOpportunityRepository(pool)
	eventRepo := repository.NewEventRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	contactRepo := repository.NewContactRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	oppHandler := handlers.NewOpportunityHandler(oppRepo, logger)
	eventHandler := handlers.NewEventHandler(eventRepo, logger)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo, logger)
	contactHandler := handlers.NewContactHandler(contactRepo, snsClient, cfg.SNSTopicARN, logger)
	userHandler := handlers.NewUserHandler(userRepo, logger)
	authHandler := handlers.NewAuthHandler(userRepo, logger)
	healthHandler := handlers.NewHealthHandler(pool)
	statusHandler := handlers.NewStatusHandler(pool)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler.Check)
	r.Route("/api", func(r chi.Router) {
		r.Use(httprate.LimitByIP(100, time.Minute))
		r.Get("/opportunities", oppHandler.Search)
		r.Get("/opportunities/{id}", oppHandler.GetByID)
		r.Get("/filters", oppHandler.GetFilters)
		r.Post("/events", eventHandler.Create)
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
				r.Get("/admin/analytics", analyticsHandler.GetAnalytics)
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

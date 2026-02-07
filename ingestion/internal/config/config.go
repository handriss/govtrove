package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

type IngestionMode string

const (
	ModeFull        IngestionMode = "full"        // CSV + API cross-reference + descriptions (default)
	ModeCSVOnly     IngestionMode = "csv-only"    // CSV download only (fast backfill)
	ModeIncremental IngestionMode = "incremental" // API-only with date filtering
)

type Config struct {
	DatabaseURL   string        `envconfig:"DATABASE_URL" required:"true"`
	SAMAPIKey     string        `envconfig:"SAM_API_KEY" required:"true"`
	AWSRegion     string        `envconfig:"AWS_REGION" default:"us-east-1"`
	SNSTopicARN   string        `envconfig:"SNS_TOPIC_ARN"`
	LogLevel      string        `envconfig:"LOG_LEVEL" default:"info"`
	LookbackDays  int           `envconfig:"LOOKBACK_DAYS" default:"7"`
	IngestionMode IngestionMode `envconfig:"INGESTION_MODE" default:"full"`

	// Development/testing options
	MockAPIURL       string `envconfig:"MOCK_API_URL"`
	RecordLimit      int    `envconfig:"RECORD_LIMIT" default:"0"`      // Limit CSV parsing (0 = no limit)
	MaxRecords       int    `envconfig:"MAX_RECORDS" default:"0"`       // Max records to insert after sorting by date (0 = no limit)
	SkipAPI          bool   `envconfig:"SKIP_API" default:"false"`
	SkipDescriptions bool   `envconfig:"SKIP_DESCRIPTIONS" default:"false"`
	VerboseLogging   bool   `envconfig:"VERBOSE_LOGGING" default:"false"`
}

func (c *Config) IsMockMode() bool {
	return c.MockAPIURL != ""
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg.IngestionMode = IngestionMode(strings.ToLower(string(cfg.IngestionMode)))
	if !cfg.IngestionMode.IsValid() {
		return nil, fmt.Errorf("invalid INGESTION_MODE: %q (must be full, csv-only, or incremental)", cfg.IngestionMode)
	}

	return &cfg, nil
}

func (m IngestionMode) IsValid() bool {
	switch m {
	case ModeFull, ModeCSVOnly, ModeIncremental:
		return true
	default:
		return false
	}
}

func (m IngestionMode) String() string {
	return string(m)
}

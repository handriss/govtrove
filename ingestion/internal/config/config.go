package config

import (
	"fmt"
	"strings"
	"time"

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

	// CSV archive settings
	S3Bucket         string `envconfig:"S3_BUCKET" default:"govtrove-data"`
	S3ArchiveEnabled bool   `envconfig:"S3_ARCHIVE_ENABLED" default:"true"`

	// Development/testing options
	MockAPIURL       string `envconfig:"MOCK_API_URL"`
	RecordLimit      int    `envconfig:"RECORD_LIMIT" default:"0"`      // Limit CSV parsing (0 = no limit)
	MaxRecords       int    `envconfig:"MAX_RECORDS" default:"0"`       // Max records to insert after sorting by date (0 = no limit)
	SkipAPI          bool   `envconfig:"SKIP_API" default:"false"`
	SkipDescriptions bool   `envconfig:"SKIP_DESCRIPTIONS" default:"false"`
	VerboseLogging   bool   `envconfig:"VERBOSE_LOGGING" default:"false"`

	// TODO(pre-launch): Remove MIN_POSTED_DATE before going live — we need full historical data.
	// This is a temporary workaround to stay within Neon free tier (512 MB).
	// Once upgraded to a paid plan, remove this env var and re-ingest all historical data.
	MinPostedDate string `envconfig:"MIN_POSTED_DATE" default:""`
}

func (c *Config) GetMinPostedDate() *time.Time {
	if c.MinPostedDate == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", c.MinPostedDate)
	if err != nil {
		return nil
	}
	return &t
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

// CSVArchiveConfig holds settings for the CSV archive service.
// Separate from Config because csvarchive doesn't need SAM_API_KEY or ingestion-specific fields.
type CSVArchiveConfig struct {
	DatabaseURL      string `envconfig:"DATABASE_URL" required:"true"`
	AWSRegion        string `envconfig:"AWS_REGION" default:"us-east-1"`
	SNSTopicARN      string `envconfig:"SNS_TOPIC_ARN"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	S3Bucket         string `envconfig:"S3_BUCKET" default:"govtrove-data"`
	S3ArchiveEnabled bool   `envconfig:"S3_ARCHIVE_ENABLED" default:"true"`
}

func LoadCSVArchive() (*CSVArchiveConfig, error) {
	var cfg CSVArchiveConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

type APIProbeConfig struct {
	DatabaseURL      string `envconfig:"DATABASE_URL" required:"true"`
	SAMAPIKey        string `envconfig:"SAM_API_KEY" required:"true"`
	AWSRegion        string `envconfig:"AWS_REGION" default:"us-east-1"`
	SNSTopicARN      string `envconfig:"SNS_TOPIC_ARN"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	Mode             string `envconfig:"MODE" default:"probe"`
	S3Bucket         string `envconfig:"S3_BUCKET" default:"govtrove-data"`
	S3ArchiveEnabled bool   `envconfig:"S3_ARCHIVE_ENABLED" default:"true"`
	LookbackDays     int    `envconfig:"LOOKBACK_DAYS" default:"7"`
}

func LoadAPIProbe() (*APIProbeConfig, error) {
	var cfg APIProbeConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
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

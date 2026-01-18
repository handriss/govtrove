package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`
	SAMAPIKey    string `envconfig:"SAM_API_KEY" required:"true"`
	AWSRegion    string `envconfig:"AWS_REGION" default:"us-east-1"`
	SNSTopicARN  string `envconfig:"SNS_TOPIC_ARN"`
	LogLevel     string `envconfig:"LOG_LEVEL" default:"info"`
	LookbackDays int    `envconfig:"LOOKBACK_DAYS" default:"1"`

	// Development/testing options
	MockAPIURL       string `envconfig:"MOCK_API_URL"`
	RecordLimit      int    `envconfig:"RECORD_LIMIT" default:"0"`
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
	return &cfg, nil
}

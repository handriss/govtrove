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
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

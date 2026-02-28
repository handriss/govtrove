package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL      string `envconfig:"DATABASE_URL" required:"true"`
	AWSRegion        string `envconfig:"AWS_REGION" default:"us-east-1"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	S3Bucket         string `envconfig:"S3_BUCKET" default:"govtrove-data"`
	S3ArchiveEnabled bool   `envconfig:"S3_ARCHIVE_ENABLED" default:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

func RequireEnv(keys ...string) error {
	var missing []string
	for _, key := range keys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}

func ParseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

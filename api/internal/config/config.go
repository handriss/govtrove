package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL    string `envconfig:"DATABASE_URL" required:"true"`
	Port           int    `envconfig:"PORT" default:"3000"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	AllowedOrigins string `envconfig:"ALLOWED_ORIGINS" default:"http://localhost:5173,http://localhost:3000"`
	SNSTopicARN    string `envconfig:"SNS_TOPIC_ARN"`
	AWSRegion      string `envconfig:"AWS_REGION" default:"us-east-1"`
	WorkOSClientID string `envconfig:"WORKOS_CLIENT_ID"`
	WorkOSAPIKey   string `envconfig:"WORKOS_API_KEY"`
	AdminEmails    string `envconfig:"ADMIN_EMAILS"`
	SentryDSN      string `envconfig:"SENTRY_DSN"`
	SESFromEmail        string `envconfig:"SES_FROM_EMAIL"`
	SESConfigSet        string `envconfig:"SES_CONFIG_SET"`
	ResendAPIKey        string `envconfig:"RESEND_API_KEY"`
	ResendFromEmail     string `envconfig:"RESEND_FROM_EMAIL"`
	ResendWebhookSecret string `envconfig:"RESEND_WEBHOOK_SECRET"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

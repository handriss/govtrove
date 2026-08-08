package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL         string `envconfig:"DATABASE_URL" required:"true"`
	Port                int    `envconfig:"PORT" default:"3000"`
	LogLevel            string `envconfig:"LOG_LEVEL" default:"info"`
	AllowedOrigins      string `envconfig:"ALLOWED_ORIGINS" default:"http://localhost:5173,http://localhost:3000"`
	SNSTopicARN         string `envconfig:"SNS_TOPIC_ARN"`
	AWSRegion           string `envconfig:"AWS_REGION" default:"us-east-1"`
	WorkOSClientID      string `envconfig:"WORKOS_CLIENT_ID"`
	WorkOSAPIKey        string `envconfig:"WORKOS_API_KEY"`
	AdminEmails         string `envconfig:"ADMIN_EMAILS"`
	SentryDSN           string `envconfig:"SENTRY_DSN"`
	SESFromEmail        string `envconfig:"SES_FROM_EMAIL"`
	SESConfigSet        string `envconfig:"SES_CONFIG_SET"`
	ResendAPIKey        string `envconfig:"RESEND_API_KEY"`
	ResendFromEmail     string `envconfig:"RESEND_FROM_EMAIL"`
	ResendWebhookSecret string `envconfig:"RESEND_WEBHOOK_SECRET"`
	StripeSecretKey     string `envconfig:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret string `envconfig:"STRIPE_WEBHOOK_SECRET"`
	StripePriceMonthly  string `envconfig:"STRIPE_PRICE_MONTHLY"`
	StripePromoCouponID string `envconfig:"STRIPE_PROMO_COUPON_ID"`
	PosthogKey          string `envconfig:"POSTHOG_KEY"`
	PosthogHost         string `envconfig:"POSTHOG_HOST" default:"https://us.i.posthog.com"`
	MCPInternalURL      string `envconfig:"MCP_INTERNAL_URL" default:"https://mcp.govtrove.com"`
	InternalAPIToken    string `envconfig:"INTERNAL_API_TOKEN"`
	OriginVerifySecret  string `envconfig:"ORIGIN_VERIFY_SECRET"`
	DSARBucket          string `envconfig:"DSAR_S3_BUCKET"`
	SearchRescueEnabled bool   `envconfig:"SEARCH_RESCUE_ENABLED" default:"false"`
	OpenRouterAPIKey    string `envconfig:"OPENROUTER_API_KEY"`
	SearchRescueModel   string `envconfig:"SEARCH_RESCUE_MODEL" default:"anthropic/claude-haiku-4.5"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

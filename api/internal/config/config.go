package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL    string `envconfig:"DATABASE_URL" required:"true"`
	Port           int    `envconfig:"PORT" default:"3000"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	AllowedOrigins string `envconfig:"ALLOWED_ORIGINS" default:"http://localhost:5173,http://localhost:3000"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

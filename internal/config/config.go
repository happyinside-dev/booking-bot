// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the application.
type Config struct {
	// Telegram
	BotToken string

	// Infrastructure
	DatabaseURL string
	RedisURL    string

	// Runtime
	AppEnv   string // "development" | "production"
	LogLevel string // "debug" | "info" | "warn" | "error"

	// Business defaults (used when creating a new business; can later be
	// overridden per-business in the businesses table).
	DefaultSlotIntervalMin  int
	DefaultMinBookingNotice int // minutes
	DefaultMaxBookingDays   int
}

// Load reads configuration from environment variables. It returns an error
// if a required variable is missing or a numeric variable is malformed.
func Load() (*Config, error) {
	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		AppEnv:      getEnvDefault("APP_ENV", "development"),
		LogLevel:    getEnvDefault("LOG_LEVEL", "info"),
	}

	var err error

	cfg.DefaultSlotIntervalMin, err = getEnvIntDefault("DEFAULT_SLOT_INTERVAL", 30)
	if err != nil {
		return nil, err
	}

	cfg.DefaultMinBookingNotice, err = getEnvIntDefault("DEFAULT_MIN_BOOKING_NOTICE", 60)
	if err != nil {
		return nil, err
	}

	cfg.DefaultMaxBookingDays, err = getEnvIntDefault("DEFAULT_MAX_BOOKING_DAYS", 30)
	if err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("config: REDIS_URL is required")
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("config: BOT_TOKEN is required")
	}

	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvIntDefault(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid int value for %s: %w", key, err)
	}
	return v, nil
}

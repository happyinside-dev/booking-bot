// Package config loads application configuration from environment variables.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the application.
type Config struct {
	// Telegram
	BotToken string
	BotBusinessID int64
	BotTokens map[string]string

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
	var err error
	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		AppEnv:      getEnvDefault("APP_ENV", "development"),
		LogLevel:    getEnvDefault("LOG_LEVEL", "info"),
	}
	if raw := os.Getenv("BOT_TOKENS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.BotTokens); err != nil {
			return nil, fmt.Errorf("config: invalid BOT_TOKENS JSON: %w", err)
		}
	}
	if raw := os.Getenv("BOT_BUSINESS_ID"); raw != "" {
		cfg.BotBusinessID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || cfg.BotBusinessID <= 0 { return nil, fmt.Errorf("config: BOT_BUSINESS_ID must be a positive integer") }
	}

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
	if len(cfg.BotTokens) == 0 && (cfg.BotToken == "" || cfg.BotBusinessID == 0) {
		return nil, fmt.Errorf("config: BOT_TOKENS or BOT_TOKEN with BOT_BUSINESS_ID is required")
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

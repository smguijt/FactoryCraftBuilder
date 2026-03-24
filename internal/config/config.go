package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	ProjectID         string
	FirebaseCredsPath string // optional: path to service account JSON; falls back to ADC
	StartingCoins     int64
	MaxOfflineSeconds int64
	DebugRoutes       bool
}

func Load() *Config {
	_ = godotenv.Load() // no-op if .env not present (Cloud Run uses env vars directly)

	return &Config{
		Port:              getEnv("PORT", "8080"),
		ProjectID:         getEnv("GCP_PROJECT_ID", ""),
		FirebaseCredsPath: getEnv("FIREBASE_CREDS_PATH", ""),
		StartingCoins:     getEnvInt64("STARTING_COINS", 500),
		MaxOfflineSeconds: getEnvInt64("MAX_OFFLINE_SECONDS", 28800), // 8 hours
		DebugRoutes:       getEnv("DEBUG_ROUTES", "true") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		slog.Warn("invalid env var, using fallback", "key", key, "value", v)
		return fallback
	}
	return n
}

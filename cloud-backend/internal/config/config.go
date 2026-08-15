package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	VPICBaseURL       string
	DTCSeedPath         string
	SuperAdminEmail    string
	SuperAdminPassword string
	CarAPIToken        string
	CarAPISecret       string
	VincarioAPIKey     string
	VincarioSecret     string
	UIDir              string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres:///mechazone?sslmode=disable"),
		VPICBaseURL:        strings.TrimRight(env("VPIC_BASE_URL", "https://vpic.nhtsa.dot.gov/api"), "/"),
		DTCSeedPath:        env("DTC_SEED_PATH", "seeds/p0xxx.csv"),
		SuperAdminEmail:    env("SUPERADMIN_EMAIL", "admin@mechazone.local"),
		SuperAdminPassword: env("SUPERADMIN_PASSWORD", "change-me-now"),
		CarAPIToken:        env("CARAPI_TOKEN", ""),
		CarAPISecret:       env("CARAPI_SECRET", ""),
		VincarioAPIKey:     env("VINCARIO_API_KEY", ""),
		VincarioSecret:     env("VINCARIO_SECRET_KEY", ""),
		UIDir:              env("UI_DIR", ""),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

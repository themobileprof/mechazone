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
	DevShopID         string
	DevTechnicianID   string
	DTCSeedPath       string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres:///mechazone?sslmode=disable"),
		VPICBaseURL:     strings.TrimRight(env("VPIC_BASE_URL", "https://vpic.nhtsa.dot.gov/api"), "/"),
		DevShopID:       env("DEV_SHOP_ID", "00000000-0000-4000-8000-000000000001"),
		DevTechnicianID: env("DEV_TECHNICIAN_ID", "00000000-0000-4000-8000-000000000002"),
		DTCSeedPath:     env("DTC_SEED_PATH", "seeds/p0xxx.csv"),
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

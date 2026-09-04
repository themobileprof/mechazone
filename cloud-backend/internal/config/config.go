// Package config loads HTTP, database, VIN, LLM, and import-dir settings from the environment.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	VPICBaseURL        string
	DTCSeedPath        string
	SuperAdminEmail    string
	SuperAdminPassword string
	CarAPIToken        string
	CarAPISecret       string
	VincarioAPIKey     string
	VincarioSecret     string
	UIDir              string
	ImportDir          string
	LLMEnabled         bool
	LLMBaseURL         string
	LLMAPIKey          string
	LLMModel           string
	EmbeddingBaseURL   string
	EmbeddingAPIKey    string
}

func Load() (Config, error) {
	loadDotEnv()
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
		ImportDir:          defaultImportDir(),
		LLMEnabled:         envBool("LLM_ENABLED", env("LLM_API_KEY", "") != ""),
		LLMBaseURL:         strings.TrimRight(env("LLM_BASE_URL", "https://api.deepseek.com"), "/"),
		LLMAPIKey:          env("LLM_API_KEY", ""),
		LLMModel:           env("LLM_MODEL", "deepseek-chat"),
		EmbeddingBaseURL:   strings.TrimRight(env("EMBEDDING_BASE_URL", "http://127.0.0.1:11434"), "/"),
		EmbeddingAPIKey:    env("EMBEDDING_API_KEY", ""),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func (c Config) LLMReady() bool {
	return c.LLMEnabled && c.LLMAPIKey != "" && c.LLMBaseURL != "" && c.LLMModel != ""
}

func (c Config) EmbeddingReady() bool {
	if c.EmbeddingBaseURL == "" {
		return false
	}
	if localEmbedHost(c.EmbeddingBaseURL) {
		return true
	}
	return c.EmbeddingAPIKey != ""
}

func localEmbedHost(u string) bool {
	u = strings.ToLower(u)
	return strings.Contains(u, "127.0.0.1") || strings.Contains(u, "localhost") || strings.Contains(u, "[::1]")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
}

func defaultImportDir() string {
	if v := env("IMPORT_DIR", ""); v != "" {
		return v
	}
	dir, err := os.Getwd()
	if err != nil {
		return "data/imported-reports"
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "docs", "project.md")); err == nil {
			return filepath.Join(dir, "data", "imported-reports")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "data/imported-reports"
}

func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 6; i++ {
		path := filepath.Join(dir, ".env")
		if f, err := os.Open(path); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
					continue
				}
				k, v, _ := strings.Cut(line, "=")
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
					v = v[1 : len(v)-1]
				}
				if os.Getenv(k) == "" {
					_ = os.Setenv(k, v)
				}
			}
			_ = f.Close()
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

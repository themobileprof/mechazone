package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mechazone/cloud-backend/internal/config"
	"mechazone/cloud-backend/internal/httpapi"
	"mechazone/cloud-backend/internal/ledger"
	"mechazone/cloud-backend/internal/vin"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := ledger.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	seedPath := cfg.DTCSeedPath
	if !filepath.IsAbs(seedPath) {
		if _, err := os.Stat(seedPath); err != nil {
			seedPath = filepath.Join("cloud-backend", cfg.DTCSeedPath)
		}
	}
	if err := store.SeedDTCs(ctx, seedPath); err != nil {
		log.Error("dtc seed", "err", err)
		os.Exit(1)
	}
	if err := store.EnsureSuperAdmin(ctx, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		log.Error("super admin", "err", err)
		os.Exit(1)
	}

	h := httpapi.New(cfg, store, vin.NewClient(cfg.VPICBaseURL), log)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

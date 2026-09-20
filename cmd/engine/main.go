// Command engine is the jevai HTTP service: it serves the web surfaces, fronts the
// Jev judgment API, and meters usage per user.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"jevai/internal/jev"
	"jevai/internal/ledger"
	"jevai/internal/server"
	"jevai/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := server.Config{
		Port:        env("PORT", "8080"),
		JevModel:    env("JEV_MODEL", "jev-latest"),
		JevKey:      os.Getenv("TYPESAFE_API_KEY"),
		TrialTokens: envInt("JEVAI_TRIAL_TOKENS", 1_000_000),
	}

	// Judge: live client when a key is set, otherwise the deterministic sample Mock.
	var judge jev.Judge = jev.Mock{}
	if cfg.JevKey != "" {
		judge = jev.New(cfg.JevKey, cfg.JevModel)
	}

	dbPath := env("JEVAI_DB", "jevai.db")

	// Ledger: durable SQLite, falling back to in-memory if the file can't be opened.
	var led ledger.Ledger
	if sq, err := ledger.OpenSQLite(dbPath); err != nil {
		log.Warn("sqlite unavailable — using in-memory ledger", "err", err)
		led = ledger.NewMemory()
	} else {
		led = sq
	}
	defer func() { _ = led.Close() }()

	// Audit store: saved, shareable results (permalink + CSV). Required for the product.
	audits, err := store.Open(dbPath)
	if err != nil {
		log.Error("audit store unavailable", "err", err)
		os.Exit(1)
	}
	defer func() { _ = audits.Close() }()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, log, judge, led, audits).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("jevai engine listening", "addr", srv.Addr, "jev_live", judge.Live())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("jevai engine stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// Command engine is the jevai HTTP service: it serves the web surfaces and
// (soon) fronts the Jev judgment API, metering usage per user.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jevai/internal/server"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := server.Config{
		Port:     env("PORT", "8080"),
		JevModel: env("JEV_MODEL", "jev-latest"),
		JevKey:   os.Getenv("TYPESAFE_API_KEY"),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(cfg, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("jevai engine listening", "addr", srv.Addr, "jev_key", cfg.JevKey != "")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown
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

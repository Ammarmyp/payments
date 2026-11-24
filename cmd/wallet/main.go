package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/config"
	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/Ammarmyp/payments/internal/platform/logging"
	"github.com/Ammarmyp/payments/internal/platform/postgres"
	"github.com/Ammarmyp/payments/internal/wallet"
)

const (
	serviceName    = "wallet"
	serviceVersion = "dev"
)

func main() {
	c := config.New(os.LookupEnv)
	logCfg := config.LogConfig{
		Level:  config.Parse(c, "LOG_LEVEL", slog.LevelInfo, logging.ParseLevel),
		Format: config.Parse(c, "LOG_FORMAT", logging.FormatText, logging.ParseFormat),
	}
	if err := c.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, err := logging.New(logging.Options{
		Writer:  os.Stdout,
		Service: serviceName,
		Version: serviceVersion,
		Level:   logCfg.Level,
		Format:  logCfg.Format,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	cfg, err := postgres.ConfigFromEnv()
	if err != nil {
		logger.Error("database config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		logger.Error("database pool", "err", err)
		os.Exit(1)
	}

	gate := wallet.NewReadyGate()
	gate.SetReady(true)

	mux := http.NewServeMux()
	wallet.NewHealthHandler(pool, gate).Register(mux)

	handler := httpx.Chain(
		mux,
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recover(logger),
	)

	srv := &http.Server{
		Addr:              ":8081",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("wallet service listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			pool.Close()
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully")

	// not-ready → drain HTTP → close pool
	gate.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}

	pool.Close()
	logger.Info("database pool closed")
}

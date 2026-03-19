// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/health"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/metrics"
	natsclient "github.com/atipparaju/nats-k8s-health-bridge/internal/nats"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/server"
)

var (
	cfgFile string
	version = "dev"
)

func main() {
	root := &cobra.Command{
		Use:     "nats-k8s-health-bridge",
		Short:   "Kubernetes health probe bridge for NATS JetStream consumers",
		Version: version,
		RunE:    run,
	}

	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to config file (default: config.yaml)")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Set up structured logger
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("starting nats-k8s-health-bridge", "version", version, "port", cfg.Server.Port)

	// Initialize Prometheus metrics registry
	reg := metrics.NewRegistry()

	// Connect to NATS
	nc, err := natsclient.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("connecting to NATS: %w", err)
	}
	defer nc.Close()

	// Create health checker
	checker := health.NewChecker(nc, cfg, reg, logger)

	// Create and start HTTP server
	srv := server.New(cfg, checker, reg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health polling loop
	checkerDone := make(chan struct{})
	go func() {
		checker.Run(ctx)
		close(checkerDone)
	}()

	// Start HTTP server
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server error: %w", err)
		}
	}

	// Graceful shutdown sequence:
	// 1. Cancel context — stops health polling loop
	// 2. Wait for checker to finish
	// 3. Shutdown HTTP server — drains in-flight requests
	// 4. Close NATS connection (via defer)
	slog.Info("shutting down gracefully")
	cancel()
	<-checkerDone
	slog.Info("health checker stopped")
	return srv.Shutdown(context.Background())
}

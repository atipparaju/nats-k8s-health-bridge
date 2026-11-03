// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/health"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/metrics"
)

// HealthChecker defines the interface for querying health state.
// This enables unit testing of HTTP handlers without a real health checker.
type HealthChecker interface {
	IsLive() bool
	IsNATSConnected() bool
	IsReady() (bool, []string)
	States() []*health.ConsumerState
	RecordHeartbeat(name string) bool
}

// Server is the HTTP server exposing probe and metrics endpoints.
type Server struct {
	srv    *http.Server
	logger *slog.Logger
	cfg    *config.Config
}

// New creates a configured HTTP server with all routes registered.
func New(cfg *config.Config, checker HealthChecker, reg *metrics.Registry, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	// /healthz — liveness: is NATS connected?
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if checker.IsLive() {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "error",
			"reason": "bridge process unavailable",
		})
	})

	// /readyz — readiness: NATS connectivity and consumer metadata availability.
	// Threshold breaches affect readiness only with explicit strict-mode opt-in.
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !checker.IsNATSConnected() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":  "not_ready",
				"reasons": []string{"not connected to NATS"},
			})
			return
		}

		ready, failures := checker.IsReady()
		if ready {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":  "not_ready",
			"reasons": failures,
		})
	})

	// Application heartbeats are opt-in and must be restricted to trusted pod-local callers.
	mux.HandleFunc("POST /heartbeat/{consumer}", func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) {
			http.Error(w, "heartbeat requires loopback caller", http.StatusForbidden)
			return
		}
		if !checker.RecordHeartbeat(r.PathValue("consumer")) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown consumer"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
	})

	// /metrics — Prometheus exposition
	mux.Handle("/metrics", protected(cfg.Server.MetricsToken, cfg.Server.AllowUnauthenticatedDiagnostics, promhttp.HandlerFor(
		reg.PrometheusRegistry,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)))

	// /status — full JSON state for debugging
	mux.Handle("/status", protected(cfg.Server.StatusToken, cfg.Server.AllowUnauthenticatedDiagnostics, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		states := checker.States()
		ready, failures := checker.IsReady()
		writeJSON(w, http.StatusOK, map[string]any{
			"nats_connected": checker.IsNATSConnected(),
			"ready":          ready,
			"failures":       failures,
			"consumers":      states,
			"timestamp":      time.Now().UTC(),
		})
	})))

	// Health of the sidecar itself
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "nats-k8s-health-bridge",
			"docs":    "/healthz /readyz /metrics /status",
		})
	})

	httpSrv := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.BindAddress, fmt.Sprintf("%d", cfg.Server.Port)),
		Handler:      loggingMiddleware(logger, mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{srv: httpSrv, logger: logger, cfg: cfg}
}

// ListenAndServe starts the HTTP server. Blocks until the server stops.
func (s *Server) ListenAndServe() error {
	s.logger.Info("HTTP server listening", "addr", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.Server.ShutdownTimeout)
	defer cancel()
	s.logger.Info("shutting down HTTP server")
	return s.srv.Shutdown(shutdownCtx)
}

// isLoopback rejects non-local heartbeat requests. X-Forwarded-For is not trusted.
func isLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil { return false }
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// protected applies optional bearer-token authentication to diagnostic endpoints.
// No token fails closed by default. Anonymous access is explicit local-test opt-in.
func protected(token string, allowAnonymous bool, next http.Handler) http.Handler {
	if token == "" && allowAnonymous { return next }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "diagnostic token required", http.StatusUnauthorized)
			return
		}
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(provided) != len(token) || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// loggingMiddleware logs each HTTP request.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		// Skip logging /metrics to avoid noise
		if r.URL.Path != "/metrics" {
			logger.Debug("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", time.Since(start).String(),
			)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

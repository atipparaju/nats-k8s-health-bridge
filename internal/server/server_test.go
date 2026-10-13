// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"io"
	"testing"
	"time"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/health"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/metrics"
)

// mockHealthChecker implements HealthChecker for testing.
type mockHealthChecker struct {
	live     bool
	connected bool
	ready    bool
	failures []string
	states   []*health.ConsumerState
	heartbeatName string
}

func (m *mockHealthChecker) IsLive() bool {
	return m.live
}

func (m *mockHealthChecker) IsNATSConnected() bool { return m.connected }

func (m *mockHealthChecker) IsReady() (bool, []string) {
	return m.ready, m.failures
}

func (m *mockHealthChecker) RecordHeartbeat(name string) bool { m.heartbeatName = name; return name == "test-consumer" }

func (m *mockHealthChecker) States() []*health.ConsumerState {
	return m.states
}

func makeRequest(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func setupServer(checker HealthChecker) http.Handler {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            8080,
			ShutdownTimeout: 10 * time.Second,
		},
	}
	reg := metrics.NewRegistry()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, checker, reg, logger)
	return srv.srv.Handler
}

func TestHealthzHandler_Connected(t *testing.T) {
	checker := &mockHealthChecker{live: true}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/healthz")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestHealthzHandler_Disconnected(t *testing.T) {
	checker := &mockHealthChecker{live: false}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/healthz")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %q, want %q", body["status"], "error")
	}
}

func TestReadyzHandler_Ready(t *testing.T) {
	checker := &mockHealthChecker{live: true, connected: true, ready: true}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/readyz")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestReadyzHandler_NotReady(t *testing.T) {
	checker := &mockHealthChecker{
		live:     true,
		connected: true,
		ready:    false,
		failures: []string{"test-consumer: degraded"},
	}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/readyz")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "not_ready" {
		t.Errorf("status = %q, want %q", body["status"], "not_ready")
	}
	reasons, ok := body["reasons"].([]any)
	if !ok || len(reasons) == 0 {
		t.Error("expected non-empty reasons array")
	}
}

func TestReadyzHandler_NATSDisconnected(t *testing.T) {
	checker := &mockHealthChecker{live: false, ready: false}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/readyz")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestStatusHandler(t *testing.T) {
	checker := &mockHealthChecker{
		live:  true,
		connected: true,
		ready: true,
		states: []*health.ConsumerState{
			{
				Name:     "test-consumer",
				Stream:   "TEST",
				Consumer: "test",
				Status:   health.StatusHealthy,
			},
		},
	}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/status")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["nats_connected"] != true {
		t.Error("expected nats_connected = true")
	}
	if body["ready"] != true {
		t.Error("expected ready = true")
	}
	consumers, ok := body["consumers"].([]any)
	if !ok || len(consumers) != 1 {
		t.Error("expected 1 consumer in status")
	}
}

func TestRootHandler(t *testing.T) {
	checker := &mockHealthChecker{live: true}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["service"] != "nats-k8s-health-bridge" {
		t.Errorf("service = %q, want %q", body["service"], "nats-k8s-health-bridge")
	}
}

func TestNotFoundHandler(t *testing.T) {
	checker := &mockHealthChecker{live: true}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/nonexistent")

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestMetricsHandler(t *testing.T) {
	checker := &mockHealthChecker{live: true}
	handler := setupServer(checker)

	rr := makeRequest(t, handler, "GET", "/metrics")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty metrics response")
	}
}

func TestContentTypeJSON(t *testing.T) {
	checker := &mockHealthChecker{live: true}
	handler := setupServer(checker)

	endpoints := []string{"/healthz", "/readyz", "/status", "/"}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			rr := makeRequest(t, handler, "GET", ep)
			ct := rr.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}
		})
	}
}

// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeTestConfig(t, `
log_level: debug
poll_interval: 5s
server:
  port: 9090
  shutdown_timeout: 15s
nats:
  url: nats://test:4222
  connect_timeout: 3s
  reconnect_wait: 1s
  max_reconnects: 10
consumers:
  - name: test-consumer
    stream: TEST_STREAM
    consumer: test-consumer
    thresholds:
      max_lag: 500
      max_ack_pending: 100
      max_redelivery_rate: 0.05
      eval_window: 30s
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 5*time.Second)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Errorf("Server.ShutdownTimeout = %v, want %v", cfg.Server.ShutdownTimeout, 15*time.Second)
	}
	if cfg.NATS.URL != "nats://test:4222" {
		t.Errorf("NATS.URL = %q, want %q", cfg.NATS.URL, "nats://test:4222")
	}
	if cfg.NATS.MaxReconnects != 10 {
		t.Errorf("NATS.MaxReconnects = %d, want %d", cfg.NATS.MaxReconnects, 10)
	}
	if len(cfg.Consumers) != 1 {
		t.Fatalf("len(Consumers) = %d, want 1", len(cfg.Consumers))
	}
	c := cfg.Consumers[0]
	if c.Name != "test-consumer" {
		t.Errorf("Consumer.Name = %q, want %q", c.Name, "test-consumer")
	}
	if c.Thresholds.MaxLag != 500 {
		t.Errorf("Thresholds.MaxLag = %d, want %d", c.Thresholds.MaxLag, 500)
	}
	if c.Thresholds.MaxRedeliveryRate != 0.05 {
		t.Errorf("Thresholds.MaxRedeliveryRate = %f, want %f", c.Thresholds.MaxRedeliveryRate, 0.05)
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://localhost:4222
consumers:
  - stream: S
    consumer: C
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.PollInterval != 10*time.Second {
		t.Errorf("default PollInterval = %v, want %v", cfg.PollInterval, 10*time.Second)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.NATS.ConnectTimeout != 5*time.Second {
		t.Errorf("default ConnectTimeout = %v, want %v", cfg.NATS.ConnectTimeout, 5*time.Second)
	}
	if cfg.NATS.MaxReconnects != -1 {
		t.Errorf("default MaxReconnects = %d, want %d", cfg.NATS.MaxReconnects, -1)
	}
	// Auto-generated name
	if cfg.Consumers[0].Name != "S/C" {
		t.Errorf("auto-generated Name = %q, want %q", cfg.Consumers[0].Name, "S/C")
	}
}

func TestLoad_MissingNATSURL(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: ""
consumers:
  - stream: S
    consumer: C
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing nats.url")
	}
}

func TestLoad_NoConsumers(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://localhost:4222
consumers: []
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty consumers")
	}
}

func TestLoad_MissingConsumerStream(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://localhost:4222
consumers:
  - consumer: C
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing stream")
	}
}

func TestLoad_MissingConsumerName(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://localhost:4222
consumers:
  - stream: S
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing consumer name")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://localhost:4222
server:
  port: 99999
consumers:
  - stream: S
    consumer: C
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	path := writeTestConfig(t, `
nats:
  url: nats://original:4222
consumers:
  - stream: S
    consumer: C
`)
	t.Setenv("NKHB_NATS_URL", "nats://overridden:4222")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.NATS.URL != "nats://overridden:4222" {
		t.Errorf("NATS.URL = %q, want %q (env var override)", cfg.NATS.URL, "nats://overridden:4222")
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent config file")
	}
}

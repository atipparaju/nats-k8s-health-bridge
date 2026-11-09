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
	if c.Thresholds.MaxAckPending != 100 {
		t.Errorf("Thresholds.MaxAckPending = %d, want %d", c.Thresholds.MaxAckPending, 100)
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

func TestLoad_ObservationAgeDefaultsAndValidation(t *testing.T) {
	path := writeTestConfig(t, "nats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxObservationAge != 30*time.Second {
		t.Fatalf("default max_observation_age=%v", cfg.MaxObservationAge)
	}
	path = writeTestConfig(t, "poll_interval: 10s\nmax_observation_age: 5s\nnats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid max_observation_age error")
	}
}

func TestLoad_NegativeHeartbeatTimeout(t *testing.T) {
	path := writeTestConfig(t, "nats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n    heartbeat_timeout: -1s\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected negative heartbeat timeout error")
	}
}

func TestLoad_InvalidBindAddress(t *testing.T) {
	path := writeTestConfig(t, "server:\n  bind_address: invalid-host\nnats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid bind address")
	}
}

// Security regression: Secret-backed env vars must be unmarshaled even when
// token keys are absent from the YAML file.
func TestLoad_EnvironmentOnlyDiagnosticTokens(t *testing.T) {
	path := writeTestConfig(t, "nats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	t.Setenv("NKHB_SERVER_METRICS_TOKEN", "metrics-from-secret")
	t.Setenv("NKHB_SERVER_STATUS_TOKEN", "status-from-secret")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.MetricsToken != "metrics-from-secret" || cfg.Server.StatusToken != "status-from-secret" {
		t.Fatal("environment-only diagnostic tokens were not loaded")
	}
}

func TestLoad_DiagnosticTokensEnvironmentOverridesYAML(t *testing.T) {
	path := writeTestConfig(t, "server:\n  metrics_token: yaml-metrics\n  status_token: yaml-status\nnats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	t.Setenv("NKHB_SERVER_METRICS_TOKEN", "env-metrics")
	t.Setenv("NKHB_SERVER_STATUS_TOKEN", "env-status")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.MetricsToken != "env-metrics" || cfg.Server.StatusToken != "env-status" {
		t.Fatal("environment tokens must override YAML tokens")
	}
}

func TestLoad_PollTimeoutDefaultAndValidation(t *testing.T) {
	path := writeTestConfig(t, "nats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PollTimeout != 8*time.Second {
		t.Fatalf("default poll_timeout=%v, want 8s", cfg.PollTimeout)
	}
	path = writeTestConfig(t, "poll_timeout: -1s\nnats:\n  url: nats://localhost:4222\nconsumers:\n  - stream: S\n    consumer: C\n")
	if _, err := Load(path); err == nil {
		t.Fatal("negative poll_timeout must be rejected")
	}
}

const minimalNATSSecurityConfig = `nats:
  url: nats://localhost:4222
consumers:
  - stream: S
    consumer: C
`

// Each case omits security keys from YAML, recreating Secret-backed
// environment-only configuration rather than relying on Viper's lookups.
func TestLoad_NATSSecuritySettingsEnvironmentOnly(t *testing.T) {
	tests := []struct {
		name                             string
		env                              map[string]string
		credentials, nkey, cert, key, ca string
	}{
		{
			name:        "credentials file",
			env:         map[string]string{"NKHB_NATS_CREDENTIALS_FILE": "/secrets/user.creds"},
			credentials: "/secrets/user.creds",
		},
		{
			name: "nkey file",
			env:  map[string]string{"NKHB_NATS_NKEY_FILE": "/secrets/nkey.seed"},
			nkey: "/secrets/nkey.seed",
		},
		{
			name: "complete client certificate pair",
			env: map[string]string{
				"NKHB_NATS_TLS_CERT": "/tls/client.crt",
				"NKHB_NATS_TLS_KEY":  "/tls/client.key",
			},
			cert: "/tls/client.crt", key: "/tls/client.key",
		},
		{
			name: "custom CA on its own",
			env:  map[string]string{"NKHB_NATS_TLS_CA": "/tls/ca.crt"},
			ca:   "/tls/ca.crt",
		},
		{
			name: "credentials plus mTLS",
			env: map[string]string{
				"NKHB_NATS_CREDENTIALS_FILE": "/secrets/user.creds",
				"NKHB_NATS_TLS_CERT":         "/tls/client.crt",
				"NKHB_NATS_TLS_KEY":          "/tls/client.key",
				"NKHB_NATS_TLS_CA":           "/tls/ca.crt",
			},
			credentials: "/secrets/user.creds",
			cert:        "/tls/client.crt", key: "/tls/client.key", ca: "/tls/ca.crt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			cfg, err := Load(writeTestConfig(t, minimalNATSSecurityConfig))
			if err != nil {
				t.Fatal(err)
			}
			got := cfg.NATS
			if got.CredentialsFile != tt.credentials || got.NKeyFile != tt.nkey ||
				got.TLSCert != tt.cert || got.TLSKey != tt.key || got.TLSCA != tt.ca {
				t.Fatalf("NATS security options not loaded from environment: got %+v", got)
			}
		})
	}
}

// Both paths must behave identically. Optional fields loaded only from YAML
// must work, and the same fields must be overridable by environment variables.
func TestLoad_NATSSecuritySettingsYAMLAndEnvironmentPrecedence(t *testing.T) {
	path := writeTestConfig(t, `nats:
  url: nats://localhost:4222
  credentials_file: /yaml/user.creds
  tls_cert: /yaml/client.crt
  tls_key: /yaml/client.key
  tls_ca: /yaml/ca.crt
consumers:
  - stream: S
    consumer: C
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NATS.CredentialsFile != "/yaml/user.creds" || cfg.NATS.TLSCert != "/yaml/client.crt" ||
		cfg.NATS.TLSKey != "/yaml/client.key" || cfg.NATS.TLSCA != "/yaml/ca.crt" {
		t.Fatalf("YAML security config was not loaded: %+v", cfg.NATS)
	}

	t.Setenv("NKHB_NATS_CREDENTIALS_FILE", "/env/user.creds")
	t.Setenv("NKHB_NATS_TLS_CERT", "/env/client.crt")
	t.Setenv("NKHB_NATS_TLS_KEY", "/env/client.key")
	t.Setenv("NKHB_NATS_TLS_CA", "/env/ca.crt")
	cfg, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NATS.CredentialsFile != "/env/user.creds" || cfg.NATS.TLSCert != "/env/client.crt" ||
		cfg.NATS.TLSKey != "/env/client.key" || cfg.NATS.TLSCA != "/env/ca.crt" {
		t.Fatalf("environment must override YAML for security fields: %+v", cfg.NATS)
	}
}

func TestLoad_NATSSecuritySettingsRejectConflicts(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		env  map[string]string
	}{
		{
			name: "YAML both identity methods",
			yaml: "  credentials_file: /secrets/user.creds\n  nkey_file: /secrets/user.nkey\n",
		},
		{
			name: "environment both identity methods",
			env: map[string]string{
				"NKHB_NATS_CREDENTIALS_FILE": "/secrets/user.creds",
				"NKHB_NATS_NKEY_FILE":        "/secrets/user.nkey",
			},
		},
		{
			name: "environment TLS certificate without key",
			env:  map[string]string{"NKHB_NATS_TLS_CERT": "/tls/client.crt"},
		},
		{
			name: "environment TLS key without certificate",
			env:  map[string]string{"NKHB_NATS_TLS_KEY": "/tls/client.key"},
		},
		{
			name: "YAML TLS certificate without key",
			yaml: "  tls_cert: /tls/client.crt\n",
		},
		{
			name: "YAML TLS key without certificate",
			yaml: "  tls_key: /tls/client.key\n",
		},
		{
			name: "YAML and environment identity conflict",
			yaml: "  credentials_file: /secrets/user.creds\n",
			env:  map[string]string{"NKHB_NATS_NKEY_FILE": "/secrets/user.nkey"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			content := "nats:\n  url: nats://localhost:4222\n" + tt.yaml +
				"consumers:\n  - stream: S\n    consumer: C\n"
			if _, err := Load(writeTestConfig(t, content)); err == nil {
				t.Fatal("incomplete or conflicting NATS security configuration must fail validation")
			}
		})
	}
}

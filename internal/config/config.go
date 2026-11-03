// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	LogLevel  string        `mapstructure:"log_level"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	PollTimeout  time.Duration `mapstructure:"poll_timeout"`
	MaxObservationAge time.Duration `mapstructure:"max_observation_age"`
	Server    ServerConfig  `mapstructure:"server"`
	NATS      NATSConfig    `mapstructure:"nats"`
	Consumers []ConsumerConfig `mapstructure:"consumers"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	// ReadinessFailOnDegraded opts into removing a pod from service on consumer threshold breaches.
	// By default, degradation is observable but does not fail Kubernetes readiness.
	ReadinessFailOnDegraded bool `mapstructure:"readiness_fail_on_degraded"`
	Port            int           `mapstructure:"port"`
	BindAddress string `mapstructure:"bind_address"`
	MetricsToken string `mapstructure:"metrics_token"`
	StatusToken string `mapstructure:"status_token"`
	AllowUnauthenticatedDiagnostics bool `mapstructure:"allow_unauthenticated_diagnostics"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL             string        `mapstructure:"url"`
	CredentialsFile string        `mapstructure:"credentials_file"`
	NKeyFile        string        `mapstructure:"nkey_file"`
	TLSCert         string        `mapstructure:"tls_cert"`
	TLSKey          string        `mapstructure:"tls_key"`
	TLSCA           string        `mapstructure:"tls_ca"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	ReconnectWait   time.Duration `mapstructure:"reconnect_wait"`
	MaxReconnects   int           `mapstructure:"max_reconnects"`
}

// ConsumerConfig defines a single JetStream consumer to monitor.
type ConsumerConfig struct {
	Name       string     `mapstructure:"name"`
	Stream     string     `mapstructure:"stream"`
	Consumer   string     `mapstructure:"consumer"`
	Thresholds Thresholds `mapstructure:"thresholds"`
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat_timeout"`
}

// Thresholds defines health threshold values for a consumer.
type Thresholds struct {
	MaxLag            uint64        `mapstructure:"max_lag"`
	MaxAckPending     uint64        `mapstructure:"max_ack_pending"`
}

// Load reads configuration from a YAML file and environment variables.
// Environment variables override YAML values.
// Prefix: NKHB_ (e.g. NKHB_NATS_URL, NKHB_SERVER_PORT)
func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("poll_interval", "10s")
	v.SetDefault("poll_timeout", "8s")
	v.SetDefault("max_observation_age", "30s")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.bind_address", "0.0.0.0")
	v.SetDefault("server.readiness_fail_on_degraded", false)
	v.SetDefault("server.allow_unauthenticated_diagnostics", false)
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.connect_timeout", "5s")
	v.SetDefault("nats.reconnect_wait", "2s")
	v.SetDefault("nats.max_reconnects", -1) // -1 = infinite

	// Config file
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/nats-k8s-health-bridge")
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file is optional when not explicitly specified.
		// When explicitly specified via --config, missing file is a hard error.
		var notFoundErr viper.ConfigFileNotFoundError
		var pathErr *os.PathError
		if cfgFile != "" {
			return nil, fmt.Errorf("reading config file %s: %w", cfgFile, err)
		} else if !errors.As(err, &notFoundErr) && !errors.As(err, &pathErr) {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		// No config file found — will rely on defaults and env vars
	}

	// Environment variable overrides: NKHB_NATS_URL, NKHB_SERVER_PORT, etc.
	v.SetEnvPrefix("NKHB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Viper AutomaticEnv resolves lookups but does not enumerate environment-only
	// keys for Unmarshal. Explicitly register security-sensitive tokens so
	// Kubernetes Secret-backed variables cannot silently leave endpoints open.
	for _, key := range []string{"server.metrics_token", "server.status_token"} {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("binding environment key %s: %w", key, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.PollInterval <= 0 { return fmt.Errorf("poll_interval must be positive") }
	if cfg.PollTimeout <= 0 { return fmt.Errorf("poll_timeout must be positive") }
	if cfg.MaxObservationAge < 0 { return fmt.Errorf("max_observation_age cannot be negative") }
	if cfg.MaxObservationAge > 0 && cfg.MaxObservationAge <= cfg.PollInterval { return fmt.Errorf("max_observation_age must exceed poll_interval") }
	if cfg.NATS.URL == "" {
		return fmt.Errorf("nats.url is required")
	}
	if cfg.Server.BindAddress == "" { return fmt.Errorf("server.bind_address must not be empty") }
	if ip := net.ParseIP(cfg.Server.BindAddress); ip == nil { return fmt.Errorf("server.bind_address must be an IP address") }
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if len(cfg.Consumers) == 0 {
		return fmt.Errorf("at least one consumer must be configured")
	}
	for i, c := range cfg.Consumers {
		if c.HeartbeatTimeout < 0 { return fmt.Errorf("consumers[%d].heartbeat_timeout cannot be negative", i) }
		if c.Stream == "" {
			return fmt.Errorf("consumers[%d].stream is required", i)
		}
		if c.Consumer == "" {
			return fmt.Errorf("consumers[%d].consumer is required", i)
		}
		if c.Name == "" {
			cfg.Consumers[i].Name = fmt.Sprintf("%s/%s", c.Stream, c.Consumer)
		}
	}
	return nil
}

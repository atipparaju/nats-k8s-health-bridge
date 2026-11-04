// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build integration

package nats

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
)

func natsURL() string {
	if url := os.Getenv("NATS_URL"); url != "" {
		return url
	}
	return nats.DefaultURL
}

func setupTestStream(t *testing.T) (jetstream.JetStream, func()) {
	t.Helper()

	nc, err := nats.Connect(natsURL())
	if err != nil {
		t.Skipf("NATS endpoint %s unavailable: %v", safeNATSEndpoints(natsURL()), safeNATSError(err))
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		t.Fatalf("creating JetStream context: %v", err)
	}

	ctx := context.Background()

	// Create test stream
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "INTEGRATION_TEST",
		Subjects: []string{"integration.>"},
		Storage:  jetstream.MemoryStorage,
	})
	if err != nil {
		nc.Close()
		t.Fatalf("creating test stream: %v", err)
	}

	// Create test consumer
	_, err = js.CreateOrUpdateConsumer(ctx, "INTEGRATION_TEST", jetstream.ConsumerConfig{
		Durable:   "test-consumer",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		nc.Close()
		t.Fatalf("creating test consumer: %v", err)
	}

	cleanup := func() {
		_ = js.DeleteStream(ctx, "INTEGRATION_TEST")
		nc.Close()
	}
	return js, cleanup
}

func TestClient_Integration_Connect(t *testing.T) {
	_, cleanup := setupTestStream(t)
	defer cleanup()

	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:            natsURL(),
			ConnectTimeout: 5 * time.Second,
			ReconnectWait:  1 * time.Second,
			MaxReconnects:  3,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("expected IsConnected() = true")
	}
}

func TestClient_Integration_GetConsumerInfo(t *testing.T) {
	js, cleanup := setupTestStream(t)
	defer cleanup()

	// Publish some messages to create lag
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_, err := js.Publish(ctx, "integration.test", []byte("test message"))
		if err != nil {
			t.Fatalf("publishing message: %v", err)
		}
	}

	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:            natsURL(),
			ConnectTimeout: 5 * time.Second,
			ReconnectWait:  1 * time.Second,
			MaxReconnects:  3,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	info, err := client.GetConsumerInfo(ctx, "INTEGRATION_TEST", "test-consumer")
	if err != nil {
		t.Fatalf("GetConsumerInfo() error: %v", err)
	}

	if info.NumPending != 10 {
		t.Errorf("NumPending = %d, want 10", info.NumPending)
	}
	if info.NumAckPending != 0 {
		t.Errorf("NumAckPending = %d, want 0", info.NumAckPending)
	}
}

func TestClient_Integration_GetConsumerInfo_NotFound(t *testing.T) {
	_, cleanup := setupTestStream(t)
	defer cleanup()

	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:            natsURL(),
			ConnectTimeout: 5 * time.Second,
			ReconnectWait:  1 * time.Second,
			MaxReconnects:  3,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	started := time.Now()
	_, err = client.GetConsumerInfo(ctx, "INTEGRATION_TEST", "nonexistent-consumer")
	if err == nil {
		t.Error("expected error for nonexistent consumer")
	}
	if elapsed := time.Since(started); elapsed >= 1900*time.Millisecond {
		t.Errorf("missing consumer should fail without backoff retries; elapsed=%v", elapsed)
	}
}

 // Test an actual NATS connection rather than only checking decoded fields:
 // missing Secret-mounted credentials must not be silently ignored in favor
 // of the anonymous server used by the integration fixture.
func TestClient_Integration_EnvironmentOnlyMissingCredentialsFailsClosed(t *testing.T) {
	_, cleanup := setupTestStream(t)
	defer cleanup()

	cfgPath := filepath.Join(t.TempDir(), "bridge.yaml")
	yaml := "nats:\n  url: " + natsURL() + "\nconsumers:\n  - stream: S\n    consumer: C\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing-user.creds")
	t.Setenv("NKHB_NATS_CREDENTIALS_FILE", missing)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NATS.CredentialsFile != missing {
		t.Fatal("expected environment-only credentials to be passed into the NATS connection")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client, err := New(cfg, logger)
	if err == nil {
		client.Close()
		t.Fatal("connection should not silently succeed using anonymous access when credentials path is invalid")
	}
}

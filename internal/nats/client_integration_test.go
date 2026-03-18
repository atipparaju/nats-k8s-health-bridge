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
		t.Skipf("NATS not available at %s: %v", natsURL(), err)
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
	_, err = client.GetConsumerInfo(ctx, "INTEGRATION_TEST", "nonexistent-consumer")
	if err == nil {
		t.Error("expected error for nonexistent consumer")
	}
}

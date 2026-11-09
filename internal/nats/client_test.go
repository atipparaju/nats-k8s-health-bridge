// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package nats

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
)

func TestShouldRetryConsumerInfo(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"consumer not found", jetstream.ErrConsumerNotFound, false},
		{"wrapped consumer not found", fmt.Errorf("loading: %w", jetstream.ErrConsumerNotFound), false},
		{"stream not found", jetstream.ErrStreamNotFound, false},
		{"authorization failure", natsgo.ErrAuthorization, false},
		{"cancelled", context.Canceled, false},
		{"request timeout is transient", context.DeadlineExceeded, true},
		{"no responders is transient", natsgo.ErrNoResponders, true},
		{"temporary error", errors.New("temporary network error"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryConsumerInfo(tt.err); got != tt.want {
				t.Fatalf("shouldRetryConsumerInfo(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// RetryOnFailedConnect should return a usable (disconnected) connection while
// the server is offline. The HTTP server can then start with readiness=503.
func TestClient_InitialNATSOutageDoesNotPreventStartup(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{NATS: config.NATSConfig{
		URL:            "nats://" + addr,
		ConnectTimeout: 200 * time.Millisecond,
		ReconnectWait:  50 * time.Millisecond,
		MaxReconnects:  -1,
	}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("temporary initial NATS outage should not fail bridge startup: %v", err)
	}
	if client.IsConnected() {
		client.Close()
		t.Fatal("offline NATS must not report connected")
	}

	done := make(chan struct{})
	go func() {
		client.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("closing an offline/reconnecting NATS connection must not block shutdown")
	}
}

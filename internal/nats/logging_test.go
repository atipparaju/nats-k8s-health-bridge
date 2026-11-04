// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package nats

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	natsgo "github.com/nats-io/nats.go"
)

// Async NATS connection callbacks may log after Close returns. The collector
// must synchronize reads with those callback writes under -race.
type synchronizedLogBuffer struct {
	mu sync.Mutex
	buf bytes.Buffer
}

func (b *synchronizedLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *synchronizedLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestSafeNATSEndpoints(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"password", "nats://alice:synthetic-password@broker.example:4222", "broker.example:4222"},
		{"token", "nats://synthetic-token@broker.example:4222", "broker.example:4222"},
		{"percent encoded", "tls://u%40name:p%2Fword@broker.example:4443", "broker.example:4443"},
		{"query and path", "nats://alice:secret@broker.example:4222/path?token=secret#fragment", "broker.example:4222"},
		{"multiple servers", "nats://user:pw@first.example:4222, tls://key@second.example:4223", "first.example:4222,second.example:4223"},
		{"bare host", "localhost:4222", "localhost:4222"},
		{"ipv6", "nats://user:secret@[::1]:4222", "[::1]:4222"},
		{"empty", "", "[redacted endpoint]"},
		{"invalid", "nats://user:secret@bad host:4222", "[redacted endpoint]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeNATSEndpoints(tt.input); got != tt.want {
				t.Fatalf("safeNATSEndpoints() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafeNATSErrorPreservesTypedErrorsWithoutExposingText(t *testing.T) {
	const secret = "synthetic-password"
	cause := fmt.Errorf("request to nats://user:%s@broker:4222 failed: %w", secret, natsgo.ErrNoServers)
	err := safeNATSError(cause)
	if !errors.Is(err, natsgo.ErrNoServers) {
		t.Fatal("redacted error must preserve errors.Is semantics")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "user") || strings.Contains(err.Error(), "broker") {
		t.Fatalf("redacted error contains connection details: %q", err.Error())
	}
	if err.Error() != "NATS server unavailable" {
		t.Fatalf("unexpected sanitized error category: %q", err.Error())
	}
	if safeNATSError(nil) != nil {
		t.Fatal("nil errors must remain nil")
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	logger.Warn("NATS disconnected", "error", err, "endpoint", safeNATSEndpoints("nats://user:"+secret+"@broker:4222"))
	if strings.Contains(logs.String(), secret) || strings.Contains(logs.String(), "user:") {
		t.Fatalf("structured log exposed credentials: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "broker:4222") {
		t.Fatal("redacted logs should preserve host metadata for diagnostics")
	}
}

func TestSafeNATSErrorUnknownCauseDoesNotExposeMessage(t *testing.T) {
	const privateText = "synthetic-api-token"
	err := safeNATSError(errors.New("transport said " + privateText))
	if err == nil || strings.Contains(err.Error(), privateText) {
		t.Fatalf("unexpected secret disclosure in error: %v", err)
	}
	if err.Error() != "NATS operation failed (details redacted)" {
		t.Fatalf("unexpected fallback error: %q", err.Error())
	}
}

// Startup resilience changes an offline first connect from a fatal error to
// a disconnected client. The reconnecting-state log must still redact userinfo.
func TestClient_InitialOfflineReconnectDoesNotLogCredentials(t *testing.T) {
	const username = "synthetic-user"
	const password = "synthetic-pass"
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:            "nats://" + username + ":" + password + "@127.0.0.1:1",
			ConnectTimeout: 200 * time.Millisecond,
			ReconnectWait:  50 * time.Millisecond,
			MaxReconnects:  -1,
		},
	}
	var logs synchronizedLogBuffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	client, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("unavailable NATS should start a reconnecting client: %v", err)
	}
	if client.IsConnected() {
		client.Close()
		t.Fatal("offline endpoint must not appear connected")
	}
	client.Close()
	output := logs.String()
	for _, sensitive := range []string{username, password, username + ":" + password} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("credential %q appeared in diagnostics", sensitive)
		}
	}
	if !strings.Contains(output, "127.0.0.1:1") {
		t.Fatalf("sanitized reconnecting-state log lost endpoint metadata: %q", output)
	}
}

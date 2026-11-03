// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package health

import (
	"context"
	"fmt"
	"log/slog"
	"io"
	"testing"
	"time"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/metrics"
	natsclient "github.com/atipparaju/nats-k8s-health-bridge/internal/nats"
)

// mockNATSClient implements NATSClient for testing.
type mockNATSClient struct {
	connected    bool
	consumerInfo *natsclient.ConsumerInfo
	err          error
}

func (m *mockNATSClient) IsConnected() bool {
	return m.connected
}

func (m *mockNATSClient) GetConsumerInfo(_ context.Context, _, _ string) (*natsclient.ConsumerInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.consumerInfo, nil
}

func newTestChecker(client NATSClient, consumers []config.ConsumerConfig) *Checker {
	cfg := &config.Config{
		PollInterval: 1 * time.Second,
		Consumers:    consumers,
	}
	reg := metrics.NewRegistry()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewChecker(client, cfg, reg, logger)
}

func defaultConsumers() []config.ConsumerConfig {
	return []config.ConsumerConfig{
		{
			Name:     "test-consumer",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxLag:        1000,
				MaxAckPending: 500,
			},
		},
	}
}

func TestNewChecker_InitializesStates(t *testing.T) {
	mock := &mockNATSClient{connected: true}
	checker := newTestChecker(mock, defaultConsumers())

	states := checker.States()
	if len(states) != 1 {
		t.Fatalf("len(states) = %d, want 1", len(states))
	}
	if states[0].Status != StatusUnknown {
		t.Errorf("initial status = %q, want %q", states[0].Status, StatusUnknown)
	}
	if states[0].Name != "test-consumer" {
		t.Errorf("name = %q, want %q", states[0].Name, "test-consumer")
	}
}

func TestChecker_IsLive(t *testing.T) {
	tests := []struct {
		name      string
		connected bool
		want      bool
	}{
		{"connected", true, true},
		{"disconnected", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockNATSClient{connected: tt.connected}
			checker := newTestChecker(mock, defaultConsumers())
			if got := checker.IsLive(); got != tt.want {
				t.Errorf("IsLive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker_Healthy(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    100,
			NumAckPending: 50,
		},
	}
	checker := newTestChecker(mock, defaultConsumers())

	ctx := context.Background()
	checker.pollAll(ctx)

	ready, failures := checker.IsReady()
	if !ready {
		t.Errorf("IsReady() = false, want true; failures: %v", failures)
	}

	states := checker.States()
	if states[0].Status != StatusHealthy {
		t.Errorf("status = %q, want %q", states[0].Status, StatusHealthy)
	}
}

func TestChecker_DegradedLag(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    1500, // exceeds max_lag of 1000
			NumAckPending: 50,
		},
	}
	checker := newTestChecker(mock, defaultConsumers())

	ctx := context.Background()
	checker.pollAll(ctx)

	ready, failures := checker.IsReady()
	if !ready || len(failures) != 0 {
		t.Errorf("degraded lag should not fail default readiness: ready=%v failures=%v", ready, failures)
	}

	states := checker.States()
	if states[0].Status != StatusDegraded {
		t.Errorf("status = %q, want %q", states[0].Status, StatusDegraded)
	}
}

func TestChecker_DegradedAckPending(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    100,
			NumAckPending: 600, // exceeds max_ack_pending of 500
		},
	}
	checker := newTestChecker(mock, defaultConsumers())

	ctx := context.Background()
	checker.pollAll(ctx)

	ready, _ := checker.IsReady()
	if !ready {
		t.Error("degraded ack pending should not fail default readiness")
	}

	states := checker.States()
	if states[0].Status != StatusDegraded {
		t.Errorf("status = %q, want %q", states[0].Status, StatusDegraded)
	}
}

func TestChecker_MultipleReasons(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    2000, // exceeds max_lag
			NumAckPending: 600,  // exceeds max_ack_pending
		},
	}
	checker := newTestChecker(mock, defaultConsumers())

	ctx := context.Background()
	checker.pollAll(ctx)

	states := checker.States()
	if len(states[0].Reasons) != 2 {
		t.Errorf("len(Reasons) = %d, want 2", len(states[0].Reasons))
	}
}

func TestChecker_UnknownOnError(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		err:       fmt.Errorf("connection refused"),
	}
	checker := newTestChecker(mock, defaultConsumers())

	ctx := context.Background()
	checker.pollAll(ctx)

	states := checker.States()
	if states[0].Status != StatusUnknown {
		t.Errorf("status = %q, want %q", states[0].Status, StatusUnknown)
	}
	if states[0].LastError == "" {
		t.Error("expected LastError to be set")
	}

	ready, _ := checker.IsReady()
	if ready {
		t.Error("IsReady() = true, want false (status unknown)")
	}
}

func TestChecker_ThresholdsDisabledWhenZero(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    999999,
			NumAckPending: 999999,
		},
	}
	consumers := []config.ConsumerConfig{
		{
			Name:     "no-thresholds",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxLag:        0, // disabled
				MaxAckPending: 0, // disabled
			},
		},
	}
	checker := newTestChecker(mock, consumers)

	ctx := context.Background()
	checker.pollAll(ctx)

	ready, _ := checker.IsReady()
	if !ready {
		t.Error("IsReady() = false, want true (thresholds disabled)")
	}
}

func TestChecker_MultipleConsumers(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:    100,
			NumAckPending: 50,
		},
	}
	consumers := []config.ConsumerConfig{
		{Name: "c1", Stream: "S1", Consumer: "c1", Thresholds: config.Thresholds{MaxLag: 1000}},
		{Name: "c2", Stream: "S2", Consumer: "c2", Thresholds: config.Thresholds{MaxLag: 1000}},
		{Name: "c3", Stream: "S3", Consumer: "c3", Thresholds: config.Thresholds{MaxLag: 1000}},
	}
	checker := newTestChecker(mock, consumers)

	ctx := context.Background()
	checker.pollAll(ctx)

	ready, _ := checker.IsReady()
	if !ready {
		t.Error("IsReady() = false, want true (all consumers healthy)")
	}

	states := checker.States()
	if len(states) != 3 {
		t.Errorf("len(states) = %d, want 3", len(states))
	}
}

func TestChecker_RecoveryFromDegraded(t *testing.T) {
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending: 2000, // exceeds threshold
		},
	}
	checker := newTestChecker(mock, defaultConsumers())
	ctx := context.Background()

	// First poll: degraded
	checker.pollAll(ctx)
	states := checker.States()
	if states[0].Status != StatusDegraded {
		t.Fatalf("expected degraded, got %q", states[0].Status)
	}

	// Lag drops below threshold
	mock.consumerInfo = &natsclient.ConsumerInfo{
		NumPending: 100,
	}

	// Second poll: recovered
	checker.pollAll(ctx)
	states = checker.States()
	if states[0].Status != StatusHealthy {
		t.Errorf("expected recovery to healthy, got %q", states[0].Status)
	}
}

func TestChecker_RunStopsOnContextCancel(t *testing.T) {
	mock := &mockNATSClient{
		connected:    true,
		consumerInfo: &natsclient.ConsumerInfo{},
	}
	cfg := &config.Config{
		PollInterval: 50 * time.Millisecond,
		Consumers:    defaultConsumers(),
	}
	reg := metrics.NewRegistry()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	checker := NewChecker(mock, cfg, reg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		checker.Run(ctx)
		close(done)
	}()

	// Let it poll a few times
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

// Strict mode preserves threshold-based readiness for deployments that opt in.
func TestChecker_StrictReadinessFailsOnDegradation(t *testing.T) {
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{NumPending: 1500}}
	checker := newTestChecker(mock, defaultConsumers())
	checker.cfg.Server.ReadinessFailOnDegraded = true
	checker.pollAll(context.Background())
	ready, failures := checker.IsReady()
	if ready || len(failures) != 1 {
		t.Fatalf("strict readiness: ready=%v failures=%v, want false and one failure", ready, failures)
	}
}

func TestChecker_DefaultReadinessPreservesDegradedStatus(t *testing.T) {
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{NumPending: 1500}}
	checker := newTestChecker(mock, defaultConsumers())
	checker.pollAll(context.Background())
	ready, failures := checker.IsReady()
	if !ready || len(failures) != 0 { t.Fatalf("ready=%v failures=%v", ready, failures) }
	states := checker.States()
	if len(states) != 1 || states[0].Status != StatusDegraded || len(states[0].Reasons) == 0 {
		t.Fatalf("degradation must remain observable: %+v", states)
	}
}

// Redelivery snapshots are diagnostic only; they must not imply a fabricated rate.
func TestChecker_RedeliverySnapshotDoesNotDegrade(t *testing.T) {
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{NumRedelivered: 500}}
	checker := newTestChecker(mock, defaultConsumers())
	checker.pollAll(context.Background())
	states := checker.States()
	if len(states) != 1 || states[0].Status != StatusHealthy || states[0].NumRedelivered != 500 {
		t.Fatalf("unexpected consumer snapshot: %+v", states)
	}
}

func TestChecker_StaleObservationFailsReadiness(t *testing.T) {
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{}}
	checker := newTestChecker(mock, defaultConsumers())
	checker.cfg.MaxObservationAge = time.Second
	checker.pollAll(context.Background())
	checker.mu.Lock()
	checker.states["test-consumer"].LastSuccessful = time.Now().Add(-2 * time.Second)
	checker.mu.Unlock()
	ready, failures := checker.IsReady()
	if ready || len(failures) == 0 { t.Fatalf("stale observation: ready=%v failures=%v", ready, failures) }
	states := checker.States()
	if states[0].Status != StatusUnknown { t.Fatalf("stale status=%s, want unknown", states[0].Status) }
	checker.pollAll(context.Background())
	ready, failures = checker.IsReady()
	if !ready { t.Fatalf("fresh observation should recover: %v", failures) }
}

func TestChecker_FailedPollDoesNotRefreshLastSuccessful(t *testing.T) {
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{}}
	checker := newTestChecker(mock, defaultConsumers())
	checker.cfg.MaxObservationAge = time.Second
	checker.pollAll(context.Background())
	original := checker.States()[0].LastSuccessful
	mock.err = fmt.Errorf("temporary failure")
	checker.pollAll(context.Background())
	state := checker.States()[0]
	if !state.LastSuccessful.Equal(original) || !state.LastChecked.After(original) { t.Fatalf("failed poll refreshed success timestamp: %+v", state) }
}

func TestChecker_ProcessingHeartbeatLifecycle(t *testing.T) {
	consumers := defaultConsumers()
	consumers[0].HeartbeatTimeout = time.Second
	mock := &mockNATSClient{connected: true, consumerInfo: &natsclient.ConsumerInfo{}}
	checker := newTestChecker(mock, consumers)
	checker.pollAll(context.Background())
	if ready, _ := checker.IsReady(); ready { t.Fatal("missing heartbeat must fail readiness") }
	if checker.RecordHeartbeat("unknown") { t.Fatal("unknown consumer accepted") }
	if !checker.RecordHeartbeat("test-consumer") { t.Fatal("heartbeat rejected") }
	if ready, failures := checker.IsReady(); !ready { t.Fatalf("fresh heartbeat: %v", failures) }
	checker.mu.Lock()
	checker.states["test-consumer"].LastHeartbeat = time.Now().Add(-2*time.Second)
	checker.mu.Unlock()
	if ready, _ := checker.IsReady(); ready { t.Fatal("expired heartbeat must fail readiness") }
	states := checker.States()
	if states[0].Status != StatusUnknown { t.Fatalf("expired heartbeat status=%s", states[0].Status) }
}

// blockingPollClient deliberately stalls one consumer until its context expires.
// The other consumer should still get a successful observation in the same cycle.
type blockingPollClient struct{}

func (blockingPollClient) IsConnected() bool { return true }
func (blockingPollClient) GetConsumerInfo(ctx context.Context, stream, consumer string) (*natsclient.ConsumerInfo, error) {
	if stream == "SLOW" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &natsclient.ConsumerInfo{}, nil
}

func TestChecker_BoundedConcurrentPollingDoesNotStarveHealthyConsumer(t *testing.T) {
	consumers := []config.ConsumerConfig{
		{Name:"slow", Stream:"SLOW", Consumer:"slow"},
		{Name:"fast", Stream:"FAST", Consumer:"fast"},
	}
	checker := newTestChecker(blockingPollClient{}, consumers)
	checker.cfg.PollTimeout = 100*time.Millisecond
	started := time.Now()
	checker.pollAll(context.Background())
	if time.Since(started) > 750*time.Millisecond { t.Fatal("polling cycle exceeded bounded latency") }
	states := checker.States()
	foundFast := false
	for _, state := range states {
		if state.Name == "fast" {
			foundFast = true
			if state.Status != StatusHealthy { t.Fatalf("fast consumer starved: %+v", state) }
		}
	}
	if !foundFast { t.Fatal("fast consumer missing from results") }
}

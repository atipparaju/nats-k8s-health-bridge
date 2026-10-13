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

func TestChecker_RedeliveryRate_Degraded(t *testing.T) {
	consumers := []config.ConsumerConfig{
		{
			Name:     "redeliver-test",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxRedeliveryRate: 0.1, // 10%
				EvalWindow:        1 * time.Minute,
			},
		},
	}

	// First poll: baseline
	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:     0,
			NumAckPending:  100,
			NumRedelivered: 5, // 5 redelivered out of 105 total
		},
	}
	checker := newTestChecker(mock, consumers)
	ctx := context.Background()
	checker.pollAll(ctx)

	// Second poll: redeliveries spike — 50 redelivered out of 100 delta = 50%
	mock.consumerInfo = &natsclient.ConsumerInfo{
		NumPending:     0,
		NumAckPending:  100,
		NumRedelivered: 55, // delta = 50 redelivered
	}
	checker.pollAll(ctx)

	states := checker.States()
	if states[0].Status != StatusDegraded {
		t.Errorf("status = %q, want %q (high redelivery rate)", states[0].Status, StatusDegraded)
	}
	foundReason := false
	for _, r := range states[0].Reasons {
		if len(r) > 0 {
			foundReason = true
		}
	}
	if !foundReason {
		t.Error("expected redelivery rate reason in degraded state")
	}
}

func TestChecker_RedeliveryRate_Healthy(t *testing.T) {
	consumers := []config.ConsumerConfig{
		{
			Name:     "redeliver-ok",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxRedeliveryRate: 0.5, // 50% — generous threshold
				EvalWindow:        1 * time.Minute,
			},
		},
	}

	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:     0,
			NumAckPending:  100,
			NumRedelivered: 5,
		},
	}
	checker := newTestChecker(mock, consumers)
	ctx := context.Background()
	checker.pollAll(ctx)

	// Small increase in redeliveries — well under 50%
	mock.consumerInfo = &natsclient.ConsumerInfo{
		NumPending:     0,
		NumAckPending:  100,
		NumRedelivered: 10, // delta = 5 redelivered, delta delivered = 5, rate = 1.0... hmm
	}
	checker.pollAll(ctx)

	// With only 5 delta redelivered and 5 delta total, rate = 1.0 which exceeds 0.5
	// Let's fix: make delta delivered much larger
	mock.consumerInfo = &natsclient.ConsumerInfo{
		NumPending:     0,
		NumAckPending:  200,
		NumRedelivered: 12, // delta from first: 7 redeliv / (200+12 - 100 - 5) = 7/107 ≈ 0.065
	}
	checker.pollAll(ctx)

	states := checker.States()
	if states[0].Status != StatusHealthy {
		t.Errorf("status = %q, want %q (low redelivery rate)", states[0].Status, StatusHealthy)
	}
}

func TestChecker_RedeliveryRate_InsufficientData(t *testing.T) {
	consumers := []config.ConsumerConfig{
		{
			Name:     "insufficient",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxRedeliveryRate: 0.01, // very low threshold
				EvalWindow:        1 * time.Minute,
			},
		},
	}

	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumPending:     0,
			NumAckPending:  100,
			NumRedelivered: 50, // high absolute number but only one sample
		},
	}
	checker := newTestChecker(mock, consumers)
	ctx := context.Background()

	// Only one poll — not enough data to compute rate
	checker.pollAll(ctx)

	states := checker.States()
	if states[0].Status != StatusHealthy {
		t.Errorf("status = %q, want %q (insufficient data for rate)", states[0].Status, StatusHealthy)
	}
}

func TestChecker_RedeliveryRate_DisabledWhenZero(t *testing.T) {
	consumers := []config.ConsumerConfig{
		{
			Name:     "disabled-rate",
			Stream:   "TEST",
			Consumer: "test",
			Thresholds: config.Thresholds{
				MaxRedeliveryRate: 0, // disabled
				EvalWindow:        1 * time.Minute,
			},
		},
	}

	mock := &mockNATSClient{
		connected: true,
		consumerInfo: &natsclient.ConsumerInfo{
			NumRedelivered: 1000,
			NumAckPending:  10,
		},
	}
	checker := newTestChecker(mock, consumers)
	ctx := context.Background()
	checker.pollAll(ctx)
	checker.pollAll(ctx)

	states := checker.States()
	if states[0].Status != StatusHealthy {
		t.Errorf("status = %q, want %q (rate check disabled)", states[0].Status, StatusHealthy)
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

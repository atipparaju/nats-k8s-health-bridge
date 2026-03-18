// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestNewRegistry_AllMetricsRegistered(t *testing.T) {
	reg := NewRegistry()

	if reg.PrometheusRegistry == nil {
		t.Fatal("PrometheusRegistry is nil")
	}
	if reg.ConsumerUp == nil {
		t.Fatal("ConsumerUp is nil")
	}
	if reg.ConsumerLag == nil {
		t.Fatal("ConsumerLag is nil")
	}
	if reg.ConsumerAckPending == nil {
		t.Fatal("ConsumerAckPending is nil")
	}
	if reg.ConsumerRedelivered == nil {
		t.Fatal("ConsumerRedelivered is nil")
	}
	if reg.NATSConnected == nil {
		t.Fatal("NATSConnected is nil")
	}
}

func TestNewRegistry_MetricsCollectable(t *testing.T) {
	reg := NewRegistry()

	// Gather all metrics — verifies they're properly registered
	families, err := reg.PrometheusRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	// Should have at least the Go runtime metrics + our custom ones
	if len(families) == 0 {
		t.Error("expected metrics families, got 0")
	}
}

func TestNewRegistry_GaugeLabels(t *testing.T) {
	reg := NewRegistry()

	// Set values with labels — should not panic
	reg.ConsumerUp.WithLabelValues("test-name", "TEST_STREAM", "test-consumer").Set(1)
	reg.ConsumerLag.WithLabelValues("test-name", "TEST_STREAM", "test-consumer").Set(42)
	reg.ConsumerAckPending.WithLabelValues("test-name", "TEST_STREAM", "test-consumer").Set(10)
	reg.ConsumerRedelivered.WithLabelValues("test-name", "TEST_STREAM", "test-consumer").Add(5)
	reg.NATSConnected.Set(1)

	// Verify we can gather without error
	families, err := reg.PrometheusRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	// Check specific metrics exist
	found := make(map[string]bool)
	for _, f := range families {
		found[f.GetName()] = true
	}

	expected := []string{
		"nats_consumer_up",
		"nats_consumer_lag",
		"nats_consumer_ack_pending",
		"nats_consumer_redelivered_total",
		"nats_connected",
	}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("metric %q not found in gathered families", name)
		}
	}
}

func TestNewRegistry_IndependentRegistries(t *testing.T) {
	reg1 := NewRegistry()
	reg2 := NewRegistry()

	// Each registry should be independent
	reg1.NATSConnected.Set(1)
	reg2.NATSConnected.Set(0)

	families1, _ := reg1.PrometheusRegistry.Gather()
	families2, _ := reg2.PrometheusRegistry.Gather()

	getValue := func(families []*dto.MetricFamily, name string) float64 {
		for _, f := range families {
			if f.GetName() == name {
				return f.GetMetric()[0].GetGauge().GetValue()
			}
		}
		return -1
	}

	if v := getValue(families1, "nats_connected"); v != 1 {
		t.Errorf("reg1 nats_connected = %f, want 1", v)
	}
	if v := getValue(families2, "nats_connected"); v != 0 {
		t.Errorf("reg2 nats_connected = %f, want 0", v)
	}
}

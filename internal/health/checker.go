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
	"sync"
	"time"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	"github.com/atipparaju/nats-k8s-health-bridge/internal/metrics"
	natsclient "github.com/atipparaju/nats-k8s-health-bridge/internal/nats"
)

// NATSClient defines the interface for querying NATS connection and consumer state.
// This enables unit testing of the health checker without a running NATS server.
type NATSClient interface {
	IsConnected() bool
	GetConsumerInfo(ctx context.Context, stream, consumer string) (*natsclient.ConsumerInfo, error)
}

// Status represents the health state of a single consumer.
type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusDegraded Status = "degraded"
	StatusUnknown  Status = "unknown"
)

// ConsumerState holds the latest evaluated state for one consumer.
type ConsumerState struct {
	Name           string    `json:"name"`
	Stream         string    `json:"stream"`
	Consumer       string    `json:"consumer"`
	Status         Status    `json:"status"`
	NumPending     uint64    `json:"num_pending"`
	NumAckPending  int       `json:"num_ack_pending"`
	NumRedelivered int       `json:"num_redelivered"`
	LastChecked    time.Time `json:"last_checked"`
	LastError      string    `json:"last_error,omitempty"`
	Reasons        []string  `json:"reasons,omitempty"`
}

// Checker polls JetStream consumer info and evaluates health.
type Checker struct {
	client           NATSClient
	cfg              *config.Config
	reg              *metrics.Registry
	logger           *slog.Logger
	mu               sync.RWMutex
	states           map[string]*ConsumerState
}

// NewChecker creates a new health checker.
func NewChecker(client NATSClient, cfg *config.Config, reg *metrics.Registry, logger *slog.Logger) *Checker {
	states := make(map[string]*ConsumerState, len(cfg.Consumers))
	for _, c := range cfg.Consumers {
		states[c.Name] = &ConsumerState{
			Name:     c.Name,
			Stream:   c.Stream,
			Consumer: c.Consumer,
			Status:   StatusUnknown,
		}
	}
	return &Checker{
		client:            client,
		cfg:               cfg,
		reg:               reg,
		logger:            logger,
		states:            states,
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()

	// Run once immediately
	c.pollAll(ctx)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("health checker stopping")
			return
		case <-ticker.C:
			c.pollAll(ctx)
		}
	}
}

func (c *Checker) pollAll(ctx context.Context) {
	for _, consumerCfg := range c.cfg.Consumers {
		c.pollOne(ctx, consumerCfg)
	}
}

func (c *Checker) pollOne(ctx context.Context, consumerCfg config.ConsumerConfig) {
	info, err := c.client.GetConsumerInfo(ctx, consumerCfg.Stream, consumerCfg.Consumer)

	c.mu.Lock()
	state := c.states[consumerCfg.Name]
	state.LastChecked = time.Now()

	if err != nil {
		state.Status = StatusUnknown
		state.LastError = err.Error()
		state.Reasons = []string{fmt.Sprintf("failed to fetch consumer info: %s", err)}
		c.logger.Warn("failed to fetch consumer info",
			"consumer", consumerCfg.Name,
			"error", err,
		)
		c.reg.ConsumerUp.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(0)
		c.mu.Unlock()
		return
	}

	// Update raw metric values
	state.NumPending = info.NumPending
	state.NumAckPending = info.NumAckPending
	state.NumRedelivered = info.NumRedelivered
	state.LastError = ""

	c.reg.ConsumerLag.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumPending))
	c.reg.ConsumerAckPending.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumAckPending))
	c.reg.ConsumerRedelivered.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumRedelivered))

	// Evaluate thresholds
	reasons := c.evaluate(info, consumerCfg.Thresholds)
	if len(reasons) == 0 {
		state.Status = StatusHealthy
		state.Reasons = nil
		c.reg.ConsumerUp.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(1)
	} else {
		state.Status = StatusDegraded
		state.Reasons = reasons
		c.reg.ConsumerUp.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(0)
		c.logger.Warn("consumer degraded",
			"consumer", consumerCfg.Name,
			"reasons", reasons,
		)
	}

	c.mu.Unlock()

	c.logger.Debug("consumer polled",
		"consumer", consumerCfg.Name,
		"status", state.Status,
		"lag", info.NumPending,
		"ack_pending", info.NumAckPending,
	)
}

func (c *Checker) evaluate(info *natsclient.ConsumerInfo, t config.Thresholds) []string {
	var reasons []string
	if t.MaxLag > 0 && info.NumPending > t.MaxLag {
		reasons = append(reasons, fmt.Sprintf("lag %d exceeds max_lag %d", info.NumPending, t.MaxLag))
	}
	if t.MaxAckPending > 0 && uint64(info.NumAckPending) > t.MaxAckPending {
		reasons = append(reasons, fmt.Sprintf("ack_pending %d exceeds max_ack_pending %d", info.NumAckPending, t.MaxAckPending))
	}
	return reasons
}

// IsNATSConnected reports upstream NATS connectivity for readiness and diagnostics.
func (c *Checker) IsNATSConnected() bool { return c.client.IsConnected() }

// IsLive reports the health of the running bridge process, not the upstream NATS service.
// A transient NATS outage must not trigger Kubernetes liveness restarts.
// The HTTP server's ability to answer this probe establishes process liveness.
func (c *Checker) IsLive() bool {
	return true
}

// IsReady reports whether monitored consumer metadata is available.
// Degraded thresholds are observational by default: backlog alone must not evict a
// functioning consumer from Kubernetes service. Operators may explicitly opt in
// to strict threshold-based readiness via server.readiness_fail_on_degraded.
func (c *Checker) IsReady() (bool, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var failures []string
	for name, state := range c.states {
		if state.Status == StatusUnknown || (c.cfg.Server.ReadinessFailOnDegraded && state.Status == StatusDegraded) {
			failures = append(failures, fmt.Sprintf("%s: %s", name, state.Status))
		}
	}
	return len(failures) == 0, failures
}

// States returns a snapshot of all consumer states (for /status endpoint).
func (c *Checker) States() []*ConsumerState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*ConsumerState, 0, len(c.states))
	for _, s := range c.states {
		copy := *s
		out = append(out, &copy)
	}
	return out
}

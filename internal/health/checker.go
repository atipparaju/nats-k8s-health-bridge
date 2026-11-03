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
	LastSuccessful time.Time `json:"last_successful,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	Reasons        []string  `json:"reasons,omitempty"`
}

// Checker polls JetStream consumer info and evaluates health.
type Checker struct {
	client           NATSClient
	cfg              *config.Config
	reg              *metrics.Registry
	evaluator        ConsumerEvaluator
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
		evaluator:         LegacyEvaluator{},
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

// Each polling cycle uses a bounded worker pool and a shared deadline. This
// prevents a failing consumer from blocking observations of healthy consumers.
const maxConcurrentPolls = 4

func (c *Checker) pollAll(ctx context.Context) {
    timeout := c.cfg.PollTimeout
    if timeout <= 0 {
        timeout = 8 * time.Second // for tests/programmatically created configurations
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    workers := len(c.cfg.Consumers)
    if workers > maxConcurrentPolls { workers = maxConcurrentPolls }
    jobs := make(chan config.ConsumerConfig)
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                select {
                case <-ctx.Done():
                    return
                case consumerCfg, ok := <-jobs:
                    if !ok { return }
                    c.pollOne(ctx, consumerCfg)
                }
            }
        }()
    }
dispatch:
    for _, consumerCfg := range c.cfg.Consumers {
        select {
        case <-ctx.Done():
            break dispatch
        case jobs <- consumerCfg:
        }
    }
    close(jobs)
    wg.Wait()
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
	state.LastSuccessful = state.LastChecked
	state.NumPending = info.NumPending
	state.NumAckPending = info.NumAckPending
	state.NumRedelivered = info.NumRedelivered
	state.LastError = ""

	c.reg.ConsumerLag.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumPending))
	c.reg.ConsumerAckPending.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumAckPending))
	c.reg.ConsumerRedelivered.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Set(float64(info.NumRedelivered))

	// Evaluate thresholds
	reasons := c.evaluator.Evaluate(info, consumerCfg.Thresholds)
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

	status := state.Status
	c.mu.Unlock()

	c.logger.Debug("consumer polled",
		"consumer", consumerCfg.Name,
		"status", status,
		"lag", info.NumPending,
		"ack_pending", info.NumAckPending,
	)
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
// RecordHeartbeat records an application-reported processing heartbeat.
// Only configured consumer names are accepted; unknown names are rejected.
func (c *Checker) RecordHeartbeat(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.states[name]
	if !ok { return false }
	state.LastHeartbeat = time.Now()
	return true
}

func (c *Checker) heartbeatExpired(name string, state *ConsumerState, now time.Time) bool {
	for _, consumer := range c.cfg.Consumers {
		if consumer.Name == name && consumer.HeartbeatTimeout > 0 {
			return state.LastHeartbeat.IsZero() || now.Sub(state.LastHeartbeat) > consumer.HeartbeatTimeout
		}
	}
	return false
}

func (c *Checker) IsReady() (bool, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var failures []string
	for name, state := range c.states {
		if state.Status == StatusUnknown || c.stale(state, time.Now()) || c.heartbeatExpired(name, state, time.Now()) || (c.cfg.Server.ReadinessFailOnDegraded && state.Status == StatusDegraded) {
			if c.heartbeatExpired(name, state, time.Now()) {
				failures = append(failures, fmt.Sprintf("%s: processing heartbeat missing or expired", name))
			} else if c.stale(state, time.Now()) {
				failures = append(failures, fmt.Sprintf("%s: stale consumer observation", name))
			} else {
				failures = append(failures, fmt.Sprintf("%s: %s", name, state.Status))
			}
		}
	}
	return len(failures) == 0, failures
}

// stale reports whether the most recent successful consumer poll is too old.
// A zero max age disables age enforcement for manually constructed configurations.
func (c *Checker) stale(state *ConsumerState, now time.Time) bool {
	return c.cfg.MaxObservationAge > 0 && (state.LastSuccessful.IsZero() || now.Sub(state.LastSuccessful) > c.cfg.MaxObservationAge)
}

// States returns a snapshot of all consumer states (for /status endpoint).
func (c *Checker) States() []*ConsumerState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*ConsumerState, 0, len(c.states))
	for name, s := range c.states {
		copy := *s
		copy.Reasons = append([]string(nil), s.Reasons...)
		if c.heartbeatExpired(name, s, time.Now()) {
			copy.Status = StatusUnknown
			copy.Reasons = append(copy.Reasons, "processing heartbeat missing or expired")
		}
		if c.stale(s, time.Now()) {
			copy.Status = StatusUnknown
			copy.Reasons = append(copy.Reasons, "consumer observation is stale")
		}
		out = append(out, &copy)
	}
	return out
}

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

// Status represents the health state of a single consumer.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// ConsumerState holds the latest evaluated state for one consumer.
type ConsumerState struct {
	Name           string    `json:"name"`
	Stream         string    `json:"stream"`
	Consumer       string    `json:"consumer"`
	Status         Status    `json:"status"`
	NumPending     uint64    `json:"num_pending"`
	NumAckPending  int       `json:"num_ack_pending"`
	NumRedelivered uint64    `json:"num_redelivered"`
	LastChecked    time.Time `json:"last_checked"`
	LastError      string    `json:"last_error,omitempty"`
	Reasons        []string  `json:"reasons,omitempty"`
}

// Checker polls JetStream consumer info and evaluates health.
type Checker struct {
	client  *natsclient.Client
	cfg     *config.Config
	reg     *metrics.Registry
	logger  *slog.Logger
	mu      sync.RWMutex
	states  map[string]*ConsumerState
}

// NewChecker creates a new health checker.
func NewChecker(client *natsclient.Client, cfg *config.Config, reg *metrics.Registry, logger *slog.Logger) *Checker {
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
		client: client,
		cfg:    cfg,
		reg:    reg,
		logger: logger,
		states: states,
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
	c.reg.ConsumerRedelivered.WithLabelValues(consumerCfg.Name, consumerCfg.Stream, consumerCfg.Consumer).Add(float64(info.NumRedelivered))

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

// IsLive returns true if NATS is connected (liveness).
func (c *Checker) IsLive() bool {
	return c.client.IsConnected()
}

// IsReady returns true if all consumers are healthy (readiness).
func (c *Checker) IsReady() (bool, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var failures []string
	for name, state := range c.states {
		if state.Status != StatusHealthy {
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

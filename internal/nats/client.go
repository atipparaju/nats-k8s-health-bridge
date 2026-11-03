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
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
)

// Client wraps a NATS connection and JetStream handle.
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *slog.Logger
}

// ConsumerInfo holds the health-relevant fields from a JetStream consumer.
type ConsumerInfo struct {
	NumPending     uint64
	NumAckPending  int
	NumRedelivered int
	IsActive       bool
}

// New creates and returns a connected NATS client.
func New(cfg *config.Config, logger *slog.Logger) (*Client, error) {
	opts := []nats.Option{
		nats.Name("nats-k8s-health-bridge"),
		nats.Timeout(cfg.NATS.ConnectTimeout),
		nats.ReconnectWait(cfg.NATS.ReconnectWait),
		nats.MaxReconnects(cfg.NATS.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
	}

	// Optional TLS
	if cfg.NATS.TLSCert != "" && cfg.NATS.TLSKey != "" {
		opts = append(opts, nats.ClientCert(cfg.NATS.TLSCert, cfg.NATS.TLSKey))
	}
	if cfg.NATS.TLSCA != "" {
		opts = append(opts, nats.RootCAs(cfg.NATS.TLSCA))
	}

	// Optional auth
	if cfg.NATS.CredentialsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.NATS.CredentialsFile))
	} else if cfg.NATS.NKeyFile != "" {
		opt, err := nats.NkeyOptionFromSeed(cfg.NATS.NKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading nkey: %w", err)
		}
		opts = append(opts, opt)
	}

	nc, err := nats.Connect(cfg.NATS.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS at %s: %w", cfg.NATS.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("creating JetStream context: %w", err)
	}

	logger.Info("connected to NATS", "url", nc.ConnectedUrl())

	return &Client{nc: nc, js: js, logger: logger}, nil
}

// IsConnected returns true if the NATS connection is currently active.
func (c *Client) IsConnected() bool {
	return c.nc.IsConnected()
}

const (
	maxRetries = 2
	retryDelay = 1 * time.Second
)

// GetConsumerInfo fetches live consumer info from JetStream with retry on transient errors.
func (c *Client) GetConsumerInfo(ctx context.Context, stream, consumer string) (*ConsumerInfo, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("retrying consumer info fetch",
				"stream", stream,
				"consumer", consumer,
				"attempt", attempt+1,
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry for %s/%s: %w", stream, consumer, ctx.Err())
			case <-time.After(retryDelay):
			}
		}

		info, err := c.fetchConsumerInfo(ctx, stream, consumer)
		if err == nil {
			return info, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !shouldRetryConsumerInfo(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// shouldRetryConsumerInfo retries only failures that might recover without a
// configuration change. Missing streams/consumers and authorization errors are
// permanent for a given poll; the next scheduled cycle still checks again.
func shouldRetryConsumerInfo(err error) bool {
	return !errors.Is(err, jetstream.ErrConsumerNotFound) &&
		!errors.Is(err, jetstream.ErrStreamNotFound) &&
		!errors.Is(err, nats.ErrAuthorization) &&
		!errors.Is(err, context.Canceled)
}

func (c *Client) fetchConsumerInfo(ctx context.Context, stream, consumer string) (*ConsumerInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cons, err := c.js.Consumer(reqCtx, stream, consumer)
	if err != nil {
		return nil, fmt.Errorf("fetching consumer %s/%s: %w", stream, consumer, err)
	}

	info, err := cons.Info(reqCtx)
	if err != nil {
		return nil, fmt.Errorf("fetching consumer info for %s/%s: %w", stream, consumer, err)
	}

	return &ConsumerInfo{
		NumPending:     info.NumPending,
		NumAckPending:  info.NumAckPending,
		NumRedelivered: info.NumRedelivered,
		IsActive:       info.PushBound || info.NumWaiting > 0 || info.NumPending > 0,
	}, nil
}

// Close drains and closes the NATS connection gracefully.
func (c *Client) Close() {
	if err := c.nc.Drain(); err != nil {
		c.logger.Warn("error draining NATS connection", "error", err)
	}
}

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
	NumRedelivered uint64
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

// GetConsumerInfo fetches live consumer info from JetStream.
func (c *Client) GetConsumerInfo(ctx context.Context, stream, consumer string) (*ConsumerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	info, err := c.js.ConsumerInfo(ctx, stream, consumer)
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

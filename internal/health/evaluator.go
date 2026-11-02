// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package health

import (
	"fmt"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	natsclient "github.com/atipparaju/nats-k8s-health-bridge/internal/nats"
)

// ConsumerEvaluator evaluates a successfully retrieved JetStream consumer
// snapshot. It does not determine Kubernetes readiness, freshness, heartbeat
// health, or NATS connection state; those remain the Checker's responsibility.
type ConsumerEvaluator interface {
	Evaluate(info *natsclient.ConsumerInfo, thresholds config.Thresholds) []string
}

// LegacyEvaluator preserves the established strict-greater-than semantics
// while upstream jsm.go compatibility is assessed independently.
type LegacyEvaluator struct{}

func (LegacyEvaluator) Evaluate(info *natsclient.ConsumerInfo, thresholds config.Thresholds) []string {
	var reasons []string
	if thresholds.MaxLag > 0 && info.NumPending > thresholds.MaxLag {
		reasons = append(reasons, fmt.Sprintf("lag %d exceeds max_lag %d", info.NumPending, thresholds.MaxLag))
	}
	if thresholds.MaxAckPending > 0 && uint64(info.NumAckPending) > thresholds.MaxAckPending {
		reasons = append(reasons, fmt.Sprintf("ack_pending %d exceeds max_ack_pending %d", info.NumAckPending, thresholds.MaxAckPending))
	}
	return reasons
}

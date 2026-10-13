// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry holds all Prometheus metric definitions for the sidecar.
type Registry struct {
	ConsumerUp          *prometheus.GaugeVec
	ConsumerLag         *prometheus.GaugeVec
	ConsumerAckPending  *prometheus.GaugeVec
	ConsumerRedelivered *prometheus.GaugeVec
	NATSConnected       prometheus.Gauge
	PrometheusRegistry  *prometheus.Registry
}

// NewRegistry creates and registers all metrics with a fresh Prometheus registry.
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	r := &Registry{
		PrometheusRegistry: reg,

		ConsumerUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "nats",
			Subsystem: "consumer",
			Name:      "up",
			Help:      "1 if the consumer is within all health thresholds, 0 otherwise.",
		}, []string{"consumer_name", "stream", "consumer"}),

		ConsumerLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "nats",
			Subsystem: "consumer",
			Name:      "lag",
			Help:      "Number of undelivered messages in the stream (num_pending).",
		}, []string{"consumer_name", "stream", "consumer"}),

		ConsumerAckPending: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "nats",
			Subsystem: "consumer",
			Name:      "ack_pending",
			Help:      "Number of messages delivered but not yet acknowledged.",
		}, []string{"consumer_name", "stream", "consumer"}),

		ConsumerRedelivered: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "nats",
			Subsystem: "consumer",
			Name:      "redelivered",
			Help:      "Current number of messages pending redelivery as reported by JetStream; snapshot, not cumulative.",
		}, []string{"consumer_name", "stream", "consumer"}),

		NATSConnected: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "nats",
			Name:      "connected",
			Help:      "1 if connected to NATS server, 0 if disconnected.",
		}),
	}

	reg.MustRegister(
		r.ConsumerUp,
		r.ConsumerLag,
		r.ConsumerAckPending,
		r.ConsumerRedelivered,
		r.NATSConnected,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return r
}

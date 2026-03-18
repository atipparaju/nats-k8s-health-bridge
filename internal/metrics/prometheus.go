package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Registry holds all Prometheus metric definitions for the sidecar.
type Registry struct {
	ConsumerUp          *prometheus.GaugeVec
	ConsumerLag         *prometheus.GaugeVec
	ConsumerAckPending  *prometheus.GaugeVec
	ConsumerRedelivered *prometheus.CounterVec
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

		ConsumerRedelivered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "nats",
			Subsystem: "consumer",
			Name:      "redelivered_total",
			Help:      "Total number of redelivered messages (cumulative).",
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
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return r
}

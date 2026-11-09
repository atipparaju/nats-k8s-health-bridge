package health

import (
	"reflect"
	"testing"

	"github.com/atipparaju/nats-k8s-health-bridge/internal/config"
	natsclient "github.com/atipparaju/nats-k8s-health-bridge/internal/nats"
)

func TestLegacyEvaluatorBoundaryParity(t *testing.T) {
	tests := []struct {
		name       string
		pending    uint64
		ack        int
		thresholds config.Thresholds
		want       []string
	}{
		{"zero thresholds disable checks", 1000, 1000, config.Thresholds{}, nil},
		{"lag exactly at threshold", 100, 0, config.Thresholds{MaxLag: 100}, nil},
		{"lag above threshold", 101, 0, config.Thresholds{MaxLag: 100}, []string{"lag 101 exceeds max_lag 100"}},
		{"ack exactly at threshold", 0, 50, config.Thresholds{MaxAckPending: 50}, nil},
		{"ack above threshold", 0, 51, config.Thresholds{MaxAckPending: 50}, []string{"ack_pending 51 exceeds max_ack_pending 50"}},
		{"both above", 101, 51, config.Thresholds{MaxLag: 100, MaxAckPending: 50}, []string{"lag 101 exceeds max_lag 100", "ack_pending 51 exceeds max_ack_pending 50"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (LegacyEvaluator{}).Evaluate(&natsclient.ConsumerInfo{
				NumPending:    tt.pending,
				NumAckPending: tt.ack,
			}, tt.thresholds)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
	"testing"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestShouldRetryConsumerInfo(t *testing.T) {
	tests := []struct {
		name string
		err error
		want bool
	}{
		{"consumer not found", jetstream.ErrConsumerNotFound, false},
		{"wrapped consumer not found", fmt.Errorf("loading: %w", jetstream.ErrConsumerNotFound), false},
		{"stream not found", jetstream.ErrStreamNotFound, false},
		{"authorization failure", natsgo.ErrAuthorization, false},
		{"cancelled", context.Canceled, false},
		{"request timeout is transient", context.DeadlineExceeded, true},
		{"no responders is transient", natsgo.ErrNoResponders, true},
		{"temporary error", errors.New("temporary network error"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryConsumerInfo(tt.err); got != tt.want {
				t.Fatalf("shouldRetryConsumerInfo(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

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
	"net/url"
	"strings"

	natsgo "github.com/nats-io/nats.go"
)

// safeNATSEndpoints keeps only network hosts. Never put a raw NATS URL in
// logs: userinfo, tokens, query parameters, and paths may be confidential.
// NATS supports comma-separated server URLs, each of which is sanitized.
func safeNATSEndpoints(raw string) string {
	const hidden = "[redacted endpoint]"
	if raw == "" {
		return hidden
	}
	endpoints := strings.Split(raw, ",")
	sanitized := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			sanitized = append(sanitized, hidden)
			continue
		}
		if !strings.Contains(endpoint, "://") {
			endpoint = "nats://" + endpoint
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" || strings.ContainsAny(parsed.Host, "@/?# ") {
			sanitized = append(sanitized, hidden)
			continue
		}
		sanitized = append(sanitized, parsed.Host)
	}
	return strings.Join(sanitized, ",")
}

// redactedNATSError exposes only an error category; NATS and transport errors
// can embed an original credential-bearing server URL. Unwrap preserves typed
// error checks without leaking the underlying text through Error()/slog.
type redactedNATSError struct {
	cause error
}

func (e redactedNATSError) Error() string {
	switch {
	case errors.Is(e.cause, natsgo.ErrAuthorization):
		return "NATS authorization failure"
	case errors.Is(e.cause, natsgo.ErrNoServers):
		return "NATS server unavailable"
	case errors.Is(e.cause, natsgo.ErrTimeout), errors.Is(e.cause, context.DeadlineExceeded):
		return "NATS request timed out"
	case errors.Is(e.cause, context.Canceled):
		return "NATS operation canceled"
	case errors.Is(e.cause, natsgo.ErrDisconnected), errors.Is(e.cause, natsgo.ErrConnectionClosed):
		return "NATS connection unavailable"
	case errors.Is(e.cause, natsgo.ErrSecureConnRequired):
		return "NATS secure connection required"
	default:
		return "NATS operation failed (details redacted)"
	}
}

func (e redactedNATSError) Unwrap() error {
	return e.cause
}

// safeNATSError must be used at every boundary where a client/transport
// failure is logged or propagated to HTTP diagnostics.
func safeNATSError(err error) error {
	if err == nil {
		return nil
	}
	return redactedNATSError{cause: err}
}

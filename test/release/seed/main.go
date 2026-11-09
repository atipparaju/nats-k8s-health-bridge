// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

// Command seed provisions disposable JetStream resources for local Docker smoke tests.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	server := flag.String("server", "nats://127.0.0.1:14222", "local NATS port-forward endpoint")
	flag.Parse()

	if err := seed(*server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func seed(server string) error {
	nc, err := nats.Connect(server, nats.Timeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connecting to local NATS: %w", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("initializing JetStream: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "LAB_EVENTS",
		Subjects: []string{"lab.events.>"},
		Storage:  jetstream.MemoryStorage,
	}); err != nil {
		return fmt.Errorf("creating smoke stream: %w", err)
	}

	if _, err := js.CreateOrUpdateConsumer(ctx, "LAB_EVENTS", jetstream.ConsumerConfig{
		Durable:   "LAB_WORKER",
		AckPolicy: jetstream.AckExplicitPolicy,
	}); err != nil {
		return fmt.Errorf("creating smoke consumer: %w", err)
	}

	if _, err := js.Publish(ctx, "lab.events.smoke", []byte("synthetic local test")); err != nil {
		return fmt.Errorf("publishing synthetic message: %w", err)
	}

	fmt.Println("Prepared disposable LAB_EVENTS/LAB_WORKER with one synthetic message")
	return nil
}

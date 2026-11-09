// Copyright 2026 Anantha Tipparaju
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

// Command seed creates a disposable stream and durable consumer for Kind tests.
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
	server := flag.String("server", "nats://127.0.0.1:14222", "NATS endpoint (port-forward)")
	flag.Parse()
	if err := seed(*server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func seed(server string) error {
	nc, err := nats.Connect(server, nats.Timeout(3*time.Second))
	if err != nil {
		return err
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: "TEST", Subjects: []string{"lab.test.>"}, Storage: jetstream.MemoryStorage,
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}
	_, err = js.CreateOrUpdateConsumer(ctx, "TEST", jetstream.ConsumerConfig{
		Durable: "worker", AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}
	if _, err = js.Publish(ctx, "lab.test.message", []byte("smoke")); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	fmt.Println("Created TEST/worker and published smoke message")
	return nil
}

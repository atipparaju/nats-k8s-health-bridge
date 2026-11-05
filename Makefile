BINARY        := nats-k8s-health-bridge
MODULE        := github.com/atipparaju/nats-k8s-health-bridge
VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE_LOCAL   := nats-k8s-health-bridge:local
GOFLAGS       := -ldflags="-s -w -X main.version=$(VERSION)" -trimpath

.PHONY: all build test lint clean docker-build release-dry-run run-local helm-lint fmt vet tidy

all: build

## build: compile the binary for the current platform
build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o bin/$(BINARY) ./cmd/sidecar

## test: run all unit tests with race detector
test:
	go test -race -count=1 -timeout=60s ./...

## fmt: format all Go source files
fmt:
	gofmt -s -w .

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (requires golangci-lint to be installed)
lint:
	golangci-lint run ./...

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf bin/

## docker-build: build a local-only image (never tags or pushes GHCR)
docker-build:
	docker build --tag $(IMAGE_LOCAL) .

## release-dry-run: verify release safeguards without Docker, credentials or network access
release-dry-run:
	python3 scripts/check_release_safety.py

## run-local: run the sidecar locally against a local NATS server
## Requires: docker run -p 4222:4222 -p 8222:8222 nats:2.15.1-alpine3.22 -js
run-local:
	NKHB_NATS_URL=nats://localhost:4222 \
	NKHB_LOG_LEVEL=debug \
		go run ./cmd/sidecar --config config.yaml

## helm-lint: lint the Helm chart
helm-lint:
	helm lint helm/nats-k8s-health-bridge

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //'

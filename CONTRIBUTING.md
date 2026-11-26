# Contributing to nats-k8s-health-bridge

Thank you for your interest in contributing! This guide will help you get started.

Coding agents and automated contributors should follow [AGENTS.md](AGENTS.md). Claude users can start with [CLAUDE.md](CLAUDE.md), which points to the same shared rules.

## Development Setup

### Prerequisites

- Go 1.26+
- Docker (for running NATS locally)
- [nats CLI](https://github.com/nats-io/natscli) (optional, for manual testing)

### Getting Started

```bash
# Clone the repo
git clone https://github.com/atipparaju/nats-k8s-health-bridge.git
cd nats-k8s-health-bridge

# Install dependencies
go mod tidy

# Build
make build

# Run tests
make test
```

## Running Tests

### Unit Tests

```bash
make test
# or
go test -race -count=1 ./...
```

### Integration Tests

Integration tests require a running NATS server with JetStream:

```bash
# Start NATS
docker run -d --name nats-test -p 4222:4222 nats:2.15.3-alpine3.22@sha256:558394a23dd3c36f2d3a64b51865fd09908010e80a542cea5e75c2af543ed62a -js

# Run integration tests
go test -race -count=1 -tags=integration ./...

# Cleanup
docker stop nats-test && docker rm nats-test
```

### Local End-to-End Testing

Use the [Docker quickstart for Windows PowerShell and Linux Bash](docs/docker-quickstart.md) to run the bridge against disposable JetStream resources with the current authentication defaults. The guide builds the image locally and provides temporary diagnostic bearer tokens. Do not expose `/status` or `/metrics` without authentication on shared networks.

For repeatable local validation, use the platform-specific smoke-test scripts in `test/release/` as documented in the quickstart. No published GHCR image is required.

## Code Standards

- Follow existing patterns in the codebase
- All errors must be wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Use structured logging with `log/slog` — no `fmt.Sprintf` in log args
- All I/O functions must accept `context.Context` as the first parameter
- No package-level globals — pass dependencies via constructors
- Protect shared state with `sync.RWMutex`
- Set `Content-Type` before `WriteHeader` in HTTP handlers
- Add the Apache 2.0 copyright header to all new Go files

## Pull Requests

1. Fork the repo and create a feature branch from `main`
2. Write tests for any new functionality
3. Ensure `make test` and `go vet ./...` pass
4. Keep commits focused — one logical change per commit
5. Write clear commit messages explaining **why**, not just what

## Reporting Issues

Only share public-safe, sanitized diagnostics. Never include credentials, tokens, real internal endpoints or unredacted configuration files.

Use [GitHub Issues](https://github.com/atipparaju/nats-k8s-health-bridge/issues) to report bugs or request features. Please include:

- Go version (`go version`)
- NATS server version
- Steps to reproduce
- Expected vs actual behavior

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

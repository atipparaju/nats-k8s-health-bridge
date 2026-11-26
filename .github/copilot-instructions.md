# GitHub Copilot repository instructions

Follow the shared [AGENTS.md](../AGENTS.md) guidance for architecture, invariants, coding standards, validation, and public-safe issue/PR content.

Before editing, consult current source and tests, `go.mod`, `README.md`, and `CONTRIBUTING.md`. Do not assume old documentation reflects currently installed dependencies.

Preserve NATS connectivity/liveness separation, conservative readiness, authentication of diagnostic endpoints, credential-safe logging, and existing metric/threshold semantics.

Keep changes scoped and reviewable. Include tests for behavior changes. Do not post private diagnostics, publish artifacts, change repository settings or merge without explicit maintainer authorization.

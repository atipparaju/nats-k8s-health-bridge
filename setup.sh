#!/usr/bin/env bash
# setup.sh — Run once to initialize the repo and push to GitHub.
# Prerequisites:
#   1. Create the private repo at https://github.com/new
#      Name: nats-k8s-health-bridge  |  Visibility: Private  |  NO readme/gitignore
#   2. Authenticate GitHub CLI: gh auth login (or have git credentials configured)
#   3. Run: bash setup.sh

set -euo pipefail

GITHUB_USER="atipparaju"
REPO="nats-k8s-health-bridge"
REMOTE="git@github.com:${GITHUB_USER}/${REPO}.git"
# Use HTTPS if SSH not set up:
# REMOTE="https://github.com/${GITHUB_USER}/${REPO}.git"

echo "🔧 Initializing git repository..."
git init
git checkout -b main

echo "📝 Adding all files..."
git add .
git commit -m "feat: initial scaffold — Phase 1 sidecar (nats-k8s-health-bridge)

- HTTP sidecar exposing /healthz, /readyz, /metrics, /status
- NATS JetStream consumer polling with threshold evaluation
- Prometheus metrics: lag, ack_pending, redelivered, connection state
- 12-factor config: YAML + NKHB_ env var overrides
- Multi-stage distroless Dockerfile (gcr.io/distroless/static-debian12)
- GitHub Actions CI: test on PR, docker push to GHCR on main
- Helm chart with sidecar injection example
- CLAUDE.md for Claude Code context

Dependencies: nats.go v1.49.0, prometheus/client_golang v1.21.1,
              cobra v1.9.1, viper v1.20.1, Go 1.23

Phase 2 (Operator + CRD) tracked in CLAUDE.md TODO section."

echo "🔗 Adding remote..."
git remote add origin "${REMOTE}"

echo "🚀 Pushing to GitHub..."
git push -u origin main

echo ""
echo "✅ Done! Repo is live at: https://github.com/${GITHUB_USER}/${REPO}"
echo ""
echo "Next steps:"
echo "  1. cd into the project directory"
echo "  2. Run: go mod tidy    (generates go.sum)"
echo "  3. Open Claude Code: claude"
echo "  4. Paste the 'Suggested First Claude Code Session Prompt' from CLAUDE.md"

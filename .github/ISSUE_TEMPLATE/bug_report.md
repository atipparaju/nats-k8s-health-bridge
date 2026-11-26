---
name: Bug Report
about: Report a bug in nats-k8s-health-bridge
title: "[Bug] "
labels: bug
assignees: ""
---

## Describe the bug

A clear description of what the bug is.

## To reproduce

Steps to reproduce the behavior:

1. Configure with `config.yaml` containing: ...
2. Run the sidecar with: ...
3. Observe: ...

## Expected behavior

What you expected to happen.

## Actual behavior

What actually happened. Include sanitized error messages or logs if available.

## Environment

- **nats-k8s-health-bridge revision:** (Git commit SHA; no supported versioned release yet)
- **Kubernetes version:** (`kubectl version`)
- **NATS server version:** (`nats-server --version`)
- **Deployment method:** (Docker / Helm / binary)
- **OS/Arch:** (e.g. linux/amd64)

## Logs

<details>
<summary>Sidecar logs</summary>

```
Paste sanitized log output only. Remove credentials, hostnames, identifiers, tokens, and private endpoint details.
```

</details>

## Additional context

Only public-safe context, screenshots, or sanitized `/status` output. Never post secrets, internal addresses or real authentication material.

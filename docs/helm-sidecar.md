# Helm-rendered sidecar example (Issue #43)

**This Helm chart creates a ConfigMap only**. It deliberately does not create a Deployment,
Pod, Service, Secret or network exposure. The rendered ConfigMap contains two data keys:
`config.yaml` (runtime config) and `sidecar-example.yaml` (a sample Pod spec fragment).
The latter is **documentation**, not an automatically injected container. Copy the
`containers[0]` and matching `volumes` entries into your own, reviewed Pod template.

## Image and pull policy

The chart defaults to `nats-k8s-health-bridge:local`, `imagePullPolicy: Never`.
Build with `make docker-build` and load that image into your chosen Kind/minikube
node before using the example. No currently accessible or public GHCR package is assumed.
For an **approved** published image, set a reviewed version or preferably an immutable
digest and set `image.pullPolicy: IfNotPresent` (or `Always` where required).
Never rely on floating `:latest` references. Check image architecture and registry auth.

```sh
helm lint helm/nats-k8s-health-bridge
helm template bridge helm/nats-k8s-health-bridge
python3 scripts/test_helm_templates.py
```

## Credentials and Secret keys

Create these **existing Kubernetes Secrets separately** in the target Pod namespace.
The chart neither creates nor checks their existence. If referenced Secrets/keys are
missing, Kubernetes will fail the container's Secret mount or env resolution
(e.g., `CreateContainerConfigError`/`FailedMount`); inspect Pod events without
printing sensitive values.

| Values key | Required key(s) in existing Secret | Runtime wiring |
|---|---|---|
| `nats.credentialsSecret` | `user.creds` | read-only volume plus `NKHB_NATS_CREDENTIALS_FILE` |
| `nats.nkeySecret` | `user.nkey` | read-only volume plus `NKHB_NATS_NKEY_FILE` |
| `nats.tlsSecret` | `ca.crt` | read-only volume plus `NKHB_NATS_TLS_CA` |
| `nats.tlsSecret` with `nats.tlsClientAuth=true` | `ca.crt`, `tls.crt`, `tls.key` | TLS CA/cert/key file environment variables |
| `server.metricsTokenSecret` | `token` | `NKHB_SERVER_METRICS_TOKEN` from `secretKeyRef` |
| `server.statusTokenSecret` | `token` | `NKHB_SERVER_STATUS_TOKEN` from `secretKeyRef` |

Choose **either** credentials or NKey (not both). Set client cert/key as a
pair; a CA-only Secret is allowed. Optional Secret mounts and token references
are omitted when corresponding values are blank. Never store tokens, private keys,
or `.creds` contents in `values.yaml` or a ConfigMap. Set
`server.allowUnauthenticatedDiagnostics=false` (the default). Both diagnostic
endpoints require tokens if used; Kubernetes `/healthz` and `/readyz` probes
remain unauthenticated.

## Probe, security and resources

The sample sets non-root UID 65532, read-only root FS, no privilege escalation,
all Linux capabilities dropped, and RuntimeDefault seccomp. Confirm the container
image and any other workloads in the Pod support this profile. Only the sidecar
container receives the shown securityContext; do not change the application container
without its own review. Keep the sidecar filesystem and Secret volumes read-only.
The default resource requests are 10m CPU/32Mi and limits 100m CPU/64Mi; size them
against actual consumer count, NATS latency, and telemetry load.

Liveness `/healthz` checks only the bridge process; NATS disconnection should
not trigger a liveness restart loop. Readiness `/readyz` reflects NATS connectivity
and consumer observation freshness. Configure network policies and scrape
authentication separately; the chart creates no network policy.

## Validation limits

`helm template` tests cover rendered paths, optional absence, and invalid secret
references (invalid names, mutually exclusive credential modes, or client TLS
without TLS Secret). They **cannot** test whether a real existing Secret/key is present,
whether the Pod has image pull access, or runtime startup/probe correctness. Validate
with a disposable cluster and actual non-production Secrets before production use.
See `docs/release-publishing.md` for separate publication authorization.

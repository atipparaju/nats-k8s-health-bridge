# Portable Docker Quickstart — Windows and Linux

This guide does not require a public GHCR package. It builds a local image and uses a private Docker network so that the bridge resolves the NATS container by name, consistently on Windows and Linux. No host-specific addresses, real credentials or workstation paths are committed.

## Prerequisites

Docker Engine/Desktop (Linux containers), Git, and the NATS CLI installed on the host. Run the commands from the repository root in PowerShell 7+ (Windows) or Bash (Linux). The demo uses memory-backed JetStream and is not a durability benchmark.

## 1. Start NATS

The following commands work in both PowerShell and Bash:

~~~sh
docker network create nats-health-demo
docker run -d --name nats-demo --network nats-health-demo -p 127.0.0.1:4222:4222 nats:2.15.1-alpine3.22 -js
nats --server nats://127.0.0.1:4222 account info
~~~

## 2. Create real JetStream resources

~~~sh
nats --server nats://127.0.0.1:4222 stream add LAB_EVENTS --subjects "lab.events.>" --storage memory --retention limits --replicas 1 --defaults
nats --server nats://127.0.0.1:4222 consumer add LAB_EVENTS LAB_WORKER --pull --ack explicit --deliver all --defaults
nats --server nats://127.0.0.1:4222 consumer info LAB_EVENTS LAB_WORKER
~~~

Expected: the LAB_WORKER durable pull consumer exists and initially has no outstanding acknowledgements.

## 3. Build the bridge

~~~sh
docker build -t nats-k8s-health-bridge:local .
~~~

Use the checked-in examples/docker/bridge-config.yaml file; unlike the generic config.yaml, its stream and consumer match the resources above.

## 4. Run the bridge (with temporary diagnostic tokens)

PowerShell:

~~~powershell
$env:NKHB_SERVER_STATUS_TOKEN = [guid]::NewGuid().ToString("N")
$env:NKHB_SERVER_METRICS_TOKEN = [guid]::NewGuid().ToString("N")
$cfgPath = Join-Path (Get-Location).Path "examples/docker/bridge-config.yaml"
docker run -d --name bridge-demo --network nats-health-demo -p 127.0.0.1:8080:8080 --env NKHB_SERVER_STATUS_TOKEN --env NKHB_SERVER_METRICS_TOKEN --mount "type=bind,source=$cfgPath,target=/etc/bridge/config.yaml,readonly" nats-k8s-health-bridge:local --config /etc/bridge/config.yaml
~~~

Linux Bash:

~~~bash
export NKHB_SERVER_STATUS_TOKEN="$(openssl rand -hex 16)"
export NKHB_SERVER_METRICS_TOKEN="$(openssl rand -hex 16)"
docker run -d --name bridge-demo --network nats-health-demo -p 127.0.0.1:8080:8080 --env NKHB_SERVER_STATUS_TOKEN --env NKHB_SERVER_METRICS_TOKEN --mount "type=bind,source=$PWD/examples/docker/bridge-config.yaml,target=/etc/bridge/config.yaml,readonly" nats-k8s-health-bridge:local --config /etc/bridge/config.yaml
~~~

Do not put real token values into Git, ConfigMaps or shared shell transcripts. The bridge listens on port 8080 within its container; the published port is restricted to the local host.

## 5. Verify the endpoints

PowerShell:

~~~powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
Invoke-RestMethod http://127.0.0.1:8080/status -Headers @{Authorization="Bearer $env:NKHB_SERVER_STATUS_TOKEN"} | ConvertTo-Json -Depth 10
(Invoke-WebRequest http://127.0.0.1:8080/metrics -Headers @{Authorization="Bearer $env:NKHB_SERVER_METRICS_TOKEN"}).StatusCode
~~~

Linux Bash:

~~~bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
curl -fsS -H "Authorization: Bearer $NKHB_SERVER_STATUS_TOKEN" http://127.0.0.1:8080/status
curl -fsS -H "Authorization: Bearer $NKHB_SERVER_METRICS_TOKEN" http://127.0.0.1:8080/metrics | head
~~~

After the first successful poll, health and readiness should be 200; the consumer should be healthy with zero pending. Test the observable pending count by publishing a message:

~~~sh
nats --server nats://127.0.0.1:4222 pub lab.events.test demo
~~~

Check authenticated status again. Missing/invalid diagnostic tokens must return HTTP 401 if the tokens are loaded correctly. Review container logs with docker logs bridge-demo and docker logs nats-demo if necessary.

## 6. Clean up

~~~sh
docker rm -f bridge-demo nats-demo
docker network rm nats-health-demo
~~~

PowerShell: Remove-Item Env:NKHB_SERVER_STATUS_TOKEN, Env:NKHB_SERVER_METRICS_TOKEN

Bash: unset NKHB_SERVER_STATUS_TOKEN NKHB_SERVER_METRICS_TOKEN

This is a local functional test only: it does not test Kubernetes, NATS persistence, TLS, application heartbeat, or multi-region failover. For those, see [local Kubernetes validation](local-kubernetes-validation.md). Docker image/package visibility must be reviewed separately before public release.

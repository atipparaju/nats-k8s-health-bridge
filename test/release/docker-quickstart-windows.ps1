#requires -Version 7.0
<#
Run manually on Windows with Docker Desktop in Linux-container mode and Go installed.
Creates disposable local containers, a network, a local image and synthetic NATS data.
Never invokes Docker login, Docker push, or GitHub publication.
#>
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
Set-Location $root
foreach ($command in @("docker", "go")) {
    if (-not (Get-Command $command -ErrorAction SilentlyContinue)) {
        throw "Missing prerequisite: $command"
    }
}

$id = [guid]::NewGuid().ToString("N").Substring(0, 10)
$network = "nkhb-47-$id"
$natsContainer = "nkhb-47-nats-$id"
$bridgeContainer = "nkhb-47-bridge-$id"
$seedExe = Join-Path ([System.IO.Path]::GetTempPath()) "nkhb-47-seed-$id.exe"
$oldStatusToken = [Environment]::GetEnvironmentVariable("NKHB_SERVER_STATUS_TOKEN", "Process")
$oldMetricsToken = [Environment]::GetEnvironmentVariable("NKHB_SERVER_METRICS_TOKEN", "Process")

function Assert-Exit([string]$task) {
    if ($LASTEXITCODE -ne 0) {
        throw "$task failed with exit code $LASTEXITCODE"
    }
}

function Assert-Unauthorized([string]$url) {
    try {
        $null = Invoke-WebRequest -Uri $url -TimeoutSec 5
        throw "Unexpected unauthenticated access to $url"
    }
    catch [Microsoft.PowerShell.Commands.HttpResponseException] {
        if ([int]$_.Exception.Response.StatusCode -ne 401) {
            throw "Unexpected HTTP status for $url"
        }
    }
}

try {
    docker network create $network | Out-Null
    Assert-Exit "Create Docker network"
    docker run -d --name $natsContainer --network $network --network-alias nats-demo -p "127.0.0.1:14222:4222" nats:2.15.1-alpine3.22 -js | Out-Null
    Assert-Exit "Start NATS container"

    go build -o $seedExe ./test/release/seed
    Assert-Exit "Build NATS seed helper"
    $seeded = $false
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        & $seedExe --server nats://127.0.0.1:14222 *> $null
        if ($LASTEXITCODE -eq 0) {
            $seeded = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $seeded) {
        throw "NATS seed failed; examine the local NATS container logs"
    }

    docker build --tag nats-k8s-health-bridge:local . | Out-Null
    Assert-Exit "Build local bridge image"

    $env:NKHB_SERVER_STATUS_TOKEN = [guid]::NewGuid().ToString("N")
    $env:NKHB_SERVER_METRICS_TOKEN = [guid]::NewGuid().ToString("N")
    $configPath = (Resolve-Path "examples/docker/bridge-config.yaml").Path
    $dockerArgs = @(
        "run", "-d", "--name", $bridgeContainer, "--network", $network,
        "-p", "127.0.0.1:18080:8080",
        "--env", "NKHB_SERVER_STATUS_TOKEN", "--env", "NKHB_SERVER_METRICS_TOKEN",
        "--mount", "type=bind,source=$configPath,target=/etc/bridge/config.yaml,readonly",
        "nats-k8s-health-bridge:local", "--config", "/etc/bridge/config.yaml"
    )
    docker @dockerArgs | Out-Null
    Assert-Exit "Start bridge container"

    $base = "http://127.0.0.1:18080"
    $ready = $false
    for ($attempt = 1; $attempt -le 40; $attempt++) {
        try {
            $response = Invoke-WebRequest -Uri "$base/readyz" -TimeoutSec 3
            if ($response.StatusCode -eq 200) {
                $ready = $true
                break
            }
        } catch {
            # Expected until first consumer observation arrives.
        }
        Start-Sleep -Seconds 1
    }
    if (-not $ready) {
        throw "Bridge failed readiness; examine the local bridge container logs"
    }
    if ((Invoke-WebRequest -Uri "$base/healthz").StatusCode -ne 200) {
        throw "Liveness endpoint failed"
    }

    Assert-Unauthorized "$base/status"
    Assert-Unauthorized "$base/metrics"
    $status = Invoke-RestMethod -Uri "$base/status" -Headers @{Authorization="Bearer $env:NKHB_SERVER_STATUS_TOKEN"}
    $metrics = Invoke-WebRequest -Uri "$base/metrics" -Headers @{Authorization="Bearer $env:NKHB_SERVER_METRICS_TOKEN"}
    if (-not $status.nats_connected -or -not $status.ready) {
        throw "Status endpoint does not report healthy NATS/readiness"
    }
    if (-not @($status.consumers | Where-Object { $_.name -eq "lab-worker" -and $_.status -eq "healthy" }).Count) {
        throw "Expected healthy LAB_WORKER consumer was not found"
    }
    if ($metrics.Content -notmatch "nats_consumer_lag" -or $metrics.Content -notmatch "nats_connected") {
        throw "Expected Prometheus metrics not found"
    }

    Write-Output "PASS: Windows Docker Desktop local smoke, real JetStream consumer, readiness and protected diagnostics."
}
finally {
    docker rm -f $bridgeContainer $natsContainer *> $null
    docker network rm $network *> $null
    Remove-Item -LiteralPath $seedExe -Force -ErrorAction SilentlyContinue
    [Environment]::SetEnvironmentVariable("NKHB_SERVER_STATUS_TOKEN", $oldStatusToken, "Process")
    [Environment]::SetEnvironmentVariable("NKHB_SERVER_METRICS_TOKEN", $oldMetricsToken, "Process")
}

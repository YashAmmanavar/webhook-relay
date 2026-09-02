# Starts everything needed to try out Webhook Relay locally:
# Postgres + Redis (Docker), the API server, the worker, and the dashboard.
# Each service opens in its own PowerShell window so their logs stay
# separate - close a window (or Ctrl+C inside it) to stop that service.

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

Write-Host "==> Starting Postgres + Redis..." -ForegroundColor Cyan
docker compose -f "$root\docker-compose.yml" up -d

Write-Host "==> Waiting for Postgres to be healthy..." -ForegroundColor Cyan
$attempts = 0
while ($true) {
    $status = docker inspect --format "{{.State.Health.Status}}" webhook-relay-postgres 2>$null
    if ($status -eq "healthy") { break }
    $attempts++
    if ($attempts -ge 30) {
        Write-Host "Postgres did not become healthy in time - continuing anyway." -ForegroundColor Yellow
        break
    }
    Start-Sleep -Seconds 1
}

if (-not (Test-Path "$root\dashboard\node_modules")) {
    Write-Host "==> Installing dashboard dependencies (first run)..." -ForegroundColor Cyan
    Push-Location "$root\dashboard"
    npm install
    Pop-Location
}

Write-Host "==> Starting API server (new window)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root'; go run ./cmd/api"

Write-Host "==> Starting worker (new window)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root'; go run ./cmd/worker"

Write-Host "==> Starting dashboard (new window)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\dashboard'; npm run dev"

Write-Host ""
Write-Host "All services starting up in separate windows:" -ForegroundColor Green
Write-Host "  API:       http://localhost:8080"
Write-Host "  Dashboard: http://localhost:5173"
Write-Host ""
Write-Host "Close each window (or Ctrl+C inside it) to stop that service." -ForegroundColor DarkGray

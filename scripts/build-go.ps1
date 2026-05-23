#!/usr/bin/env pwsh
# scripts/build-go.ps1 — Build all Go services and report failures

$env:PATH = "$env:USERPROFILE\scoop\shims;$env:USERPROFILE\scoop\apps\git\current\cmd;$env:USERPROFILE\go\bin;$env:PATH"
$env:GOPATH = "$env:USERPROFILE\go"

$services = @(
    "ingestion-agent",
    "orchestrator-agent",
    "oversight-agent",
    "messaging-agent",
    "mobile-interaction",
    "audit-agent",
    "recovery-agent"
)

$root = "C:\Users\l\OneDrive\Documents\MEDUSA\services"
$pass = 0
$fail = 0
$failures = @()

foreach ($svc in $services) {
    $dir = Join-Path $root $svc
    Write-Host "`n==> [$svc] go mod tidy..." -ForegroundColor Cyan
    Push-Location $dir

    $tidyOut = & go mod tidy 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [TIDY FAIL] $svc" -ForegroundColor Red
        $tidyOut | Select-Object -Last 8 | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkRed }
        $fail++
        $failures += "${svc}:tidy"
        Pop-Location
        continue
    }
    Write-Host "  [TIDY OK]" -ForegroundColor Green

    Write-Host "==> [$svc] go build..." -ForegroundColor Cyan
    $buildOut = & go build ./... 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [BUILD FAIL] $svc" -ForegroundColor Red
        $buildOut | Select-Object -Last 20 | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkRed }
        $fail++
        $failures += "${svc}:build"
    } else {
        Write-Host "  [BUILD OK]" -ForegroundColor Green
        $pass++
    }

    Pop-Location
}

Write-Host "`n================================================" -ForegroundColor White
Write-Host "  GO BUILD RESULTS: $pass passed, $fail failed" -ForegroundColor White
if ($failures.Count -gt 0) {
    Write-Host "  FAILURES: $($failures -join ', ')" -ForegroundColor Red
}
Write-Host "================================================`n" -ForegroundColor White
exit $fail

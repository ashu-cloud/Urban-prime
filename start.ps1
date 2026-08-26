# Urban Prime - 1-Command Full Stack PowerShell Launcher
Clear-Host
Write-Host "===================================================================" -ForegroundColor Cyan
Write-Host "   🚕 Urban Prime Mobility Platform - 1-Command Full Stack Launcher" -ForegroundColor Cyan
Write-Host "===================================================================" -ForegroundColor Cyan
Write-Host ""

# 1. Ensure .env exists
if (-not (Test-Path .env)) {
    Write-Host "[1/3] Setting up environment configuration (.env)..." -ForegroundColor Yellow
    Copy-Item .env.example .env
} else {
    Write-Host "[1/3] .env configuration found." -ForegroundColor Green
}

# 2. Launch Everything in Docker (Infra + 6 Go Services + Next.js Frontend)
Write-Host "[2/3] Building & launching entire stack in Docker..." -ForegroundColor Yellow
Write-Host "      (Runs all services in Docker - zero multiple firewall popups!)" -ForegroundColor Gray
Write-Host ""

try {
    docker compose -f deploy/docker-compose.yml up --build -d
} catch {
    docker-compose -f deploy/docker-compose.yml up --build -d
}

# 3. Open Browser after short initialization wait
Write-Host ""
Write-Host "[3/3] Opening Urban Prime in browser..." -ForegroundColor Yellow
Start-Sleep -Seconds 6
Start-Process "http://localhost:3000"

Write-Host ""
Write-Host "===================================================================" -ForegroundColor Green
Write-Host "  ✅ ALL SERVICES ARE RUNNING IN A SINGLE DOCKER CONTAINER STACK!" -ForegroundColor Green
Write-Host "     - Web App:        http://localhost:3000" -ForegroundColor White
Write-Host "     - Rider Portal:   http://localhost:3000/rider" -ForegroundColor White
Write-Host "     - Driver Cockpit: http://localhost:3000/driver" -ForegroundColor White
Write-Host "     - APISIX Gateway: http://localhost:9080" -ForegroundColor White
Write-Host "     - Jaeger Tracing: http://localhost:16686" -ForegroundColor White
Write-Host "===================================================================" -ForegroundColor Green
Write-Host ""
Write-Host "To tail live logs:   docker compose -f deploy/docker-compose.yml logs -f" -ForegroundColor Gray
Write-Host "To stop everything:  .\stop.ps1" -ForegroundColor Gray

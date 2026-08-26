Write-Host "Stopping Urban Prime Docker containers..." -ForegroundColor Yellow
try {
    docker compose -f deploy/docker-compose.yml down
} catch {
    docker-compose -f deploy/docker-compose.yml down
}
Write-Host "All services stopped." -ForegroundColor Green

# PowerShell runner for Urban Prime microservices
Write-Host "Starting Urban Prime Go Microservices..." -ForegroundColor Cyan

Start-Process wt -ArgumentList "new-tab", "-d", "Services\auth-service", "go", "run", "cmd/main.go" -ErrorAction SilentlyContinue
if (-not $?) {
    # Fallback to standard start-process cmd if Windows Terminal is not default
    Start-Process cmd -ArgumentList "/k", "cd Services\auth-service && go run cmd/main.go"
    Start-Process cmd -ArgumentList "/k", "cd Services\trip-service && go run cmd/main.go"
    Start-Process cmd -ArgumentList "/k", "cd Services\driver-service && go run cmd/main.go"
    Start-Process cmd -ArgumentList "/k", "cd Services\location-service && go run cmd/main.go"
    Start-Process cmd -ArgumentList "/k", "cd Services\payment-service && go run cmd/main.go"
    Start-Process cmd -ArgumentList "/k", "cd Services\notification-service && go run cmd/main.go"
}

Write-Host "All 6 Go Microservices are running!" -ForegroundColor Green

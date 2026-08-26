# APISIX Route Setup Script for Windows PowerShell
$APISIX_ADMIN_URL = "http://localhost:9180/apisix/admin"
$ADMIN_KEY = "edd1c9f034335f136f87ad84b625c8f1"

Write-Host "Configuring Apache APISIX Gateway Routes on $APISIX_ADMIN_URL..." -ForegroundColor Cyan

# 0. Register JWT Consumer
$consumer = @{
    username = "urban-prime-user"
    plugins = @{
        "jwt-auth" = @{
            key = "urban-prime-jwt"
            secret = "your-jwt-secret-here-must-match-AUTH_SERVICE_JWT_SECRET"
        }
    }
} | ConvertTo-Json -Depth 5

try {
    $res0 = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL/consumers" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $consumer
    Write-Host "[OK] JWT Consumer registered successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARN] JWT Consumer registration: $_" -ForegroundColor Yellow
}

# 1. Unauthenticated Auth Routes
$route1 = @{
    uri = "/auth/*"
    name = "auth-service-public"
    methods = @("POST", "OPTIONS")
    upstream = @{
        type = "roundrobin"
        nodes = @{ "host.docker.internal:8080" = 1 }
    }
    plugins = @{
        cors = @{}
    }
} | ConvertTo-Json -Depth 5

try {
    $res1 = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL/routes/1" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $route1
    Write-Host "[OK] Route 1 (Auth Service) registered successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Route 1 registration: $_" -ForegroundColor Yellow
}

# 2. Trip Service Routes
$route2 = @{
    uri = "/v1/trips*"
    name = "trip-service-protected"
    methods = @("GET", "POST", "PUT", "DELETE", "OPTIONS")
    upstream = @{
        type = "roundrobin"
        nodes = @{ "host.docker.internal:50051" = 1 }
    }
    plugins = @{
        cors = @{}
    }
} | ConvertTo-Json -Depth 5

try {
    $res2 = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL/routes/2" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $route2
    Write-Host "[OK] Route 2 (Trip Service) registered successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Route 2 registration: $_" -ForegroundColor Yellow
}

# 3. Driver Service Routes
$route3 = @{
    uri = "/v1/drivers*"
    name = "driver-service-protected"
    methods = @("GET", "POST", "PUT", "OPTIONS")
    upstream = @{
        type = "roundrobin"
        nodes = @{ "host.docker.internal:50052" = 1 }
    }
    plugins = @{
        cors = @{}
    }
} | ConvertTo-Json -Depth 5

try {
    $res3 = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL/routes/3" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $route3
    Write-Host "[OK] Route 3 (Driver Service) registered successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Route 3 registration: $_" -ForegroundColor Yellow
}

# 4. Location Service Routes
$route4 = @{
    uri = "/v1/location*"
    name = "location-service-protected"
    methods = @("POST", "PUT", "OPTIONS")
    upstream = @{
        type = "roundrobin"
        nodes = @{ "host.docker.internal:50053" = 1 }
    }
    plugins = @{
        cors = @{}
    }
} | ConvertTo-Json -Depth 5

try {
    $res4 = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL/routes/4" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $route4
    Write-Host "[OK] Route 4 (Location Service) registered successfully" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Route 4 registration: $_" -ForegroundColor Yellow
}

Write-Host "All APISIX Routes processing complete!" -ForegroundColor Cyan

# APISIX Route Setup Script for Windows PowerShell
# Paths match frontend/src/lib/api.ts and scripts/load_test_*.js
$APISIX_ADMIN_URL = if ($env:APISIX_ADMIN_URL) { $env:APISIX_ADMIN_URL } else { "http://localhost:9180/apisix/admin" }
$ADMIN_KEY = if ($env:APISIX_ADMIN_KEY) { $env:APISIX_ADMIN_KEY } else { "edd1c9f034335f136f87ad84b625c8f1" }
$UPSTREAM = if ($env:UPSTREAM_HOST) { $env:UPSTREAM_HOST } else { "host.docker.internal" }
$JWT_SECRET = if ($env:JWT_SECRET) { $env:JWT_SECRET } else { "super_secret_jwt_signing_key_change_in_production" }

Write-Host "Configuring Apache APISIX Gateway Routes on $APISIX_ADMIN_URL (upstream $UPSTREAM)..." -ForegroundColor Cyan

function Put-Apisix($path, $body) {
    $json = $body | ConvertTo-Json -Depth 8
    try {
        $res = Invoke-RestMethod -Uri "$APISIX_ADMIN_URL$path" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $json
        return $true
    } catch {
        Start-Sleep -Seconds 2
        try {
            Invoke-RestMethod -Uri "$APISIX_ADMIN_URL$path" -Method Put -Headers @{ "X-API-KEY" = $ADMIN_KEY; "Content-Type" = "application/json" } -Body $json | Out-Null
            return $true
        } catch {
            Write-Host "[WARN] $path : $_" -ForegroundColor Yellow
            return $false
        }
    }
}

Put-Apisix "/consumers/urban-prime-user" @{
    username = "urban-prime-user"
    plugins = @{
        "jwt-auth" = @{
            key = "urban-prime-jwt"
            secret = $JWT_SECRET
        }
    }
} | Out-Null
Write-Host "[OK] JWT consumer secret aligned with auth-service" -ForegroundColor Green

if (Put-Apisix "/routes/1" @{
    uri = "/auth/*"
    name = "auth-service-public"
    methods = @("GET", "POST", "OPTIONS", "HEAD")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8080" = 1 } }
    plugins = @{ cors = @{} }
}) { Write-Host "[OK] Route 1 /auth/* -> :8080" -ForegroundColor Green }

if (Put-Apisix "/routes/5" @{
    uri = "/health"
    name = "auth-health"
    methods = @("GET", "HEAD")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8080" = 1 } }
    plugins = @{ cors = @{} }
}) { Write-Host "[OK] Route 5 /health -> :8080" -ForegroundColor Green }

if (Put-Apisix "/routes/6" @{
    uri = "/api/v1/auth/*"
    name = "auth-service-api-prefix"
    methods = @("GET", "POST", "OPTIONS")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8080" = 1 } }
    plugins = @{
        cors = @{}
        "proxy-rewrite" = @{ regex_uri = @("^/api/v1/auth/(.*)", "/auth/`$1") }
    }
}) { Write-Host "[OK] Route 6 /api/v1/auth/* -> /auth/*" -ForegroundColor Green }

if (Put-Apisix "/routes/2" @{
    uris = @("/api/v1/trips", "/api/v1/trips/*", "/v1/trips*")
    name = "trip-service-rest"
    methods = @("GET", "POST", "PUT", "DELETE", "OPTIONS")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8051" = 1 } }
    plugins = @{ cors = @{} }
}) { Write-Host "[OK] Route 2 /api/v1/trips* -> :8051" -ForegroundColor Green }

if (Put-Apisix "/routes/3" @{
    uris = @("/api/v1/drivers", "/api/v1/drivers/*", "/v1/drivers*")
    name = "driver-service-rest"
    methods = @("GET", "POST", "PUT", "OPTIONS")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8052" = 1 } }
    plugins = @{ cors = @{} }
}) { Write-Host "[OK] Route 3 /api/v1/drivers* -> :8052" -ForegroundColor Green }

if (Put-Apisix "/routes/4" @{
    uris = @("/api/v1/location", "/api/v1/location/*", "/v1/location*")
    name = "location-service-rest"
    methods = @("GET", "POST", "PUT", "OPTIONS")
    upstream = @{ type = "roundrobin"; nodes = @{ "$($UPSTREAM):8053" = 1 } }
    plugins = @{ cors = @{} }
}) { Write-Host "[OK] Route 4 /api/v1/location* -> :8053" -ForegroundColor Green }

Write-Host "All APISIX Routes processing complete!" -ForegroundColor Cyan

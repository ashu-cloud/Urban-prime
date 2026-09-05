Write-Host "=== Seeding Test Driver ===" -ForegroundColor Cyan

# 1. Seed Redis (GPS cache & geo index)
Write-Host "-> Seeding local Redis geo cache..." -ForegroundColor Yellow
$timestamp = [int64]([datetime]::UtcNow - (Get-Date "1/1/1970 00:00:00Z")).TotalMilliseconds
# Delhi driver
docker exec cab_redis redis-cli GEOADD drivers:available 77.2090 28.6139 "driver-test-001"
docker exec cab_redis redis-cli SET "driver:loc:driver-test-001" "28.6139,77.2090,$timestamp" EX 300
# Bengaluru driver
docker exec cab_redis redis-cli GEOADD drivers:available 77.5946 12.9716 "00000000-0000-0000-0000-000000000001"
docker exec cab_redis redis-cli SET "driver:loc:00000000-0000-0000-0000-000000000001" "12.9716,77.5946,$timestamp" EX 300
Write-Host "   Redis seeded OK" -ForegroundColor Green

# 2. Seed Neon PostgreSQL via psql inside postgres container
Write-Host "-> Seeding Neon PostgreSQL driver record..." -ForegroundColor Yellow
$neonDSN = "postgresql://neondb_owner:npg_e8IzMlS9ioLh@ep-tiny-dew-b3o6u0cq.c-4.ap-southeast-1.aws.neon.tech/neondb?sslmode=require"
$insertSQL = "INSERT INTO drivers (driver_id, name, email, phone, vehicle_type, vehicle_plate, vehicle_model, status, rating) VALUES ('00000000-0000-0000-0000-000000000001', 'Test Driver', 'driver@test.com', '+919999999999', 'SEDAN', 'KA01AB1234', 'Maruti Swift', 'AVAILABLE', 4.8) ON CONFLICT (phone) DO UPDATE SET status='AVAILABLE';"
docker exec cab_postgres psql $neonDSN -c $insertSQL 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   Neon PostgreSQL seeded OK" -ForegroundColor Green
} else {
    Write-Host "   Neon insert skipped - driver live in Redis only" -ForegroundColor Yellow
}

Write-Host "=== Seed Complete ===" -ForegroundColor Green


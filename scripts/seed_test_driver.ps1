Write-Host "Seeding test driver into Redis..." -ForegroundColor Cyan
docker exec cab_redis redis-cli GEOADD drivers:available 77.2090 28.6139 "driver-test-001"
$timestamp = [int64]([datetime]::UtcNow - (Get-Date "1/1/1970 00:00:00Z")).TotalMilliseconds
docker exec cab_redis redis-cli SET "driver:loc:driver-test-001" "28.6139,77.2090,$timestamp" EX 300

# Also insert driver profile into PostgreSQL so DB lookup doesn't fail
docker exec cab_postgres psql -U cab_user -d cab_booking_db -c "INSERT INTO drivers (id, name, email, phone, vehicle_type, status, rating) VALUES ('driver-test-001', 'Test Driver', 'driver@test.com', '+919999999999', 'SEDAN', 'AVAILABLE', 4.8) ON CONFLICT (id) DO NOTHING;"

Write-Host "Test driver seeded." -ForegroundColor Green

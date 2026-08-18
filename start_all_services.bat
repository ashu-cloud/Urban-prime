@echo off
echo Starting Urban Prime Go Microservices...

start "Auth Service (Port 8080/50056)" cmd /k "cd Services\auth-service && go run cmd\main.go"
start "Trip Service (Port 50051)" cmd /k "cd Services\trip-service && go run cmd\main.go"
start "Driver Service (Port 50052)" cmd /k "cd Services\driver-service && go run cmd\main.go"
start "Location Service (Port 50053)" cmd /k "cd Services\location-service && go run cmd\main.go"
start "Payment Service (Port 50054)" cmd /k "cd Services\payment-service && go run cmd\main.go"
start "Notification Service (Kafka -> Centrifugo)" cmd /k "cd Services\notification-service && go run cmd\main.go"

echo All 6 Go Microservices have been launched in separate terminal windows.

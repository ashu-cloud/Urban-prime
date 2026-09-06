@echo off
TITLE Urban Prime - 1-Command Full Stack Launcher
cls
echo ===================================================================
echo    Urban Prime Mobility Platform - Full Stack Launcher
echo ===================================================================
echo.

if not exist .env (
    echo [*] Setting up environment configuration (.env)...
    copy .env.example .env >nul
)

echo [*] Starting all microservices and frontend in Docker...
echo.
docker compose up -d --build

echo.
echo [*] Waiting 10 seconds for APISIX API Gateway to initialize...
timeout /t 10 >nul

echo [*] Dynamically wiring microservice routes to APISIX Gateway...
powershell -ExecutionPolicy Bypass -File "deploy\apisix\setup_routes.ps1"

echo.
echo [*] Opening Urban Prime at http://localhost:3000...
timeout /t 5 >nul
start http://localhost:3000

echo.
echo ===================================================================
echo  ALL SERVICES ARE RUNNING!
echo     - Web App:        http://localhost:3000
echo     - Rider Portal:   http://localhost:3000/rider
echo     - Driver Cockpit: http://localhost:3000/driver
echo     - APISIX Gateway: http://localhost:9080
echo     - Jaeger Tracing: http://localhost:16686
echo ===================================================================
echo.
echo To view live logs:    docker compose logs -f (or make logs)
echo To stop everything:   docker compose down   (or make stop)
echo.
pause

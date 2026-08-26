@echo off
TITLE Urban Prime - 1-Command Full Stack Launcher
cls
echo ===================================================================
echo    🚕 Urban Prime Mobility Platform - 1-Command Full Stack Launcher
echo ===================================================================
echo.

:: 1. Ensure .env exists
if not exist .env (
    echo [1/3] Setting up environment configuration (.env)...
    copy .env.example .env >nul
) else (
    echo [1/3] .env configuration found.
)

:: 2. Launch Everything in Docker (Infra + 6 Go Services + Next.js Frontend)
echo [2/3] Building and starting all services in Docker containers...
echo       (No individual firewall prompts will be required)
echo.
docker compose -f deploy/docker-compose.yml up --build -d
if %errorlevel% neq 0 (
    docker-compose -f deploy/docker-compose.yml up --build -d
)

:: 3. Open Browser
echo.
echo [3/3] Opening Urban Prime at http://localhost:3000...
timeout /t 6 >nul
start http://localhost:3000

echo.
echo ===================================================================
echo  ✅ ALL SERVICES ARE RUNNING IN A SINGLE DOCKER CONTAINER STACK!
echo     - Web App:        http://localhost:3000
echo     - Rider Portal:   http://localhost:3000/rider
echo     - Driver Cockpit: http://localhost:3000/driver
echo     - APISIX Gateway: http://localhost:9080
echo     - Jaeger Tracing: http://localhost:16686
echo ===================================================================
echo.
echo To view live logs:    docker compose -f deploy/docker-compose.yml logs -f
echo To stop everything:   stop.bat
echo.
pause

@echo off
TITLE Urban Prime - Teardown
cls
echo Stopping Docker containers...
docker compose -f deploy/docker-compose.yml down
if %errorlevel% neq 0 (
    docker-compose -f deploy/docker-compose.yml down
)
echo Done! All services stopped.
pause

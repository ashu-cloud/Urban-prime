@echo off
TITLE Urban Prime - Teardown
cls
echo [*] Stopping all Urban Prime containers...
docker compose down
echo.
echo [*] Done! All services stopped cleanly.
pause

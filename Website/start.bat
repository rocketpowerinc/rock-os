@echo off
title ROCKOS LIVE SERVER

cd /d "%~dp0"

if not exist static-web-server.exe (
    echo static-web-server.exe missing.
    pause
    exit
)

start powershell -ExecutionPolicy Bypass -File generate-index.ps1


set /p LOCALIP=Enter your local IP address (e.g., 192.168.1.2) or leave blank for 127.0.0.1: 
if "%LOCALIP%"=="" set LOCALIP=127.0.0.1
start http://%LOCALIP%:8000

static-web-server.exe ^
--host 0.0.0.0 ^
--port 8000 ^
--root .

pause

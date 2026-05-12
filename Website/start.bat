@echo off
title ROCKOS LIVE SERVER

cd /d "%~dp0"

if not exist static-web-server.exe (
    echo static-web-server.exe missing.
    pause
    exit
)

start powershell -ExecutionPolicy Bypass -File generate-index.ps1

start http://127.0.0.1:8000

static-web-server.exe ^
--host 127.0.0.1 ^
--port 8000 ^
--root .

pause

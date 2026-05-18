@echo off
setlocal enabledelayedexpansion

set "PORT=%~1"
if "%PORT%"=="" set "PORT=8000"

echo Looking for Rock-OS on port %PORT%...

set "FOUND="
set "STOPPED_PIDS= "

for /f "tokens=5" %%P in ('netstat -ano ^| findstr /R /C:":%PORT% .*LISTENING"') do (
    set "PID=%%P"
    set "FOUND=1"
    echo !STOPPED_PIDS! | findstr /C:" !PID! " >nul
    if errorlevel 1 (
        echo Stopping process !PID! on port %PORT%...
        taskkill /PID !PID! /F >nul
        if errorlevel 1 (
            echo Failed to stop process !PID!.
            exit /b 1
        )
        set "STOPPED_PIDS=!STOPPED_PIDS!!PID! "
    )
)

if not defined FOUND (
    echo No process is listening on port %PORT%.
    exit /b 0
)

echo Rock-OS stop request complete.

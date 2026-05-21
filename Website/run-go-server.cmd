@echo off
setlocal

rem Source-only launcher for development or troubleshooting when you want to run
rem the Go server from the local source instead of using or downloading a release
rem binary. This builds a visible local dev binary first because some Windows
rem Application Control policies block the hidden executable created by go run.

cd /d "%~dp0"

set "ROCK_OS_HOST=127.0.0.1"
if /I "%~1"=="lan" set "ROCK_OS_HOST=local"
if /I "%~1"=="local" set "ROCK_OS_HOST=local"
if /I "%~1"=="all" set "ROCK_OS_HOST=local"
if /I "%~1"=="0.0.0.0" set "ROCK_OS_HOST=0.0.0.0"
if /I "%~1"=="127.0.0.1" set "ROCK_OS_HOST=127.0.0.1"

set "GOCACHE=%CD%\.gocache"
set "DEV_BINARY=rock-os-wiki-dev.exe"

echo Building Rock-OS from Go source...
go build -o "%DEV_BINARY%" .
if errorlevel 1 (
    echo Failed to build Rock-OS from source.
    echo If Go is installed correctly, check the build output above.
    echo.
    pause
    exit /b 1
)

echo Starting Rock-OS from local dev binary...
"%CD%\%DEV_BINARY%" --host "%ROCK_OS_HOST%"

echo.
pause

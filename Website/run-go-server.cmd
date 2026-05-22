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
set "ROCK_OS_WEBSITE=%CD%"
set "ROCK_OS_SOURCE=%~dp0..\cmd\rock-os-wiki"

echo Building Rock-OS from Go source...
pushd "%ROCK_OS_SOURCE%"
go build -o "%ROCK_OS_WEBSITE%\%DEV_BINARY%" .
set "ROCK_OS_BUILD_EXIT=%ERRORLEVEL%"
popd
if not "%ROCK_OS_BUILD_EXIT%"=="0" (
    echo Failed to build Rock-OS from source.
    echo If Go is installed correctly, check the build output above.
    echo.
    pause
    exit /b 1
)

echo Starting Rock-OS from local dev binary...
"%CD%\%DEV_BINARY%" --site-root "%ROCK_OS_WEBSITE%" --host "%ROCK_OS_HOST%"

echo.
pause

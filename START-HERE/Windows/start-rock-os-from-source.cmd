@echo off
setlocal

rem Source-only launcher for development or troubleshooting when you want to run
rem the Go server from the local source instead of using or downloading a release
rem binary. This tries a visible local dev binary first, then falls back to go run
rem with workspace-local Go temp folders if Windows Application Control blocks it.

for %%I in ("%~dp0..\..") do set "ROCK_OS_ROOT=%%~fI"
set "ROCK_OS_WEBSITE=%ROCK_OS_ROOT%\Website"
set "ROCK_OS_SOURCE=%ROCK_OS_ROOT%\cmd\rock-os"

cd /d "%ROCK_OS_WEBSITE%"

set "ROCK_OS_HOST=127.0.0.1"
if /I "%~1"=="lan" set "ROCK_OS_HOST=local"
if /I "%~1"=="local" set "ROCK_OS_HOST=local"
if /I "%~1"=="all" set "ROCK_OS_HOST=local"
if /I "%~1"=="0.0.0.0" set "ROCK_OS_HOST=0.0.0.0"
if /I "%~1"=="127.0.0.1" set "ROCK_OS_HOST=127.0.0.1"

set "GOCACHE=%CD%\.gocache"
set "GOTMPDIR=%CD%\.gotmp"
set "DEV_BINARY=rock-os-dev.exe"
if not exist "%GOCACHE%" mkdir "%GOCACHE%" >nul 2>nul
if not exist "%GOTMPDIR%" mkdir "%GOTMPDIR%" >nul 2>nul
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
set "ROCK_OS_EXIT=%ERRORLEVEL%"

if not "%ROCK_OS_EXIT%"=="0" (
    echo.
    echo Local dev binary could not start. Trying go run fallback...
    pushd "%ROCK_OS_SOURCE%"
    go run . --site-root "%ROCK_OS_WEBSITE%" --host "%ROCK_OS_HOST%"
    set "ROCK_OS_EXIT=%ERRORLEVEL%"
    popd
)

echo.
pause
exit /b %ROCK_OS_EXIT%

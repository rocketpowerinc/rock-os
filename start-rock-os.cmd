@echo off
setlocal

cd /d "%~dp0Website"

set "ROCK_OS_BINARY="

if exist ".\rock-os-wiki-windows-amd64.exe" (
    set "ROCK_OS_BINARY=.\rock-os-wiki-windows-amd64.exe"
) else (
    for /f "delims=" %%F in ('dir /b /o:-n ".\rock-os-wiki-v*-windows-amd64.exe" 2^>nul') do (
        if not defined ROCK_OS_BINARY set "ROCK_OS_BINARY=.\%%F"
    )
)

if defined ROCK_OS_BINARY (
    echo Starting Rock-OS from release binary...
    "%ROCK_OS_BINARY%" --host local
) else (
    echo Release binary not found. Starting Rock-OS from Go source...
    go run . --host local
)

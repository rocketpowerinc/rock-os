@echo off
setlocal

cd /d "%~dp0Website"

if exist ".\rock-os-wiki-v1.0-windows-amd64.exe" (
    echo Starting Rock-OS from release binary...
    ".\rock-os-wiki-v1.0-windows-amd64.exe" --host local
) else (
    echo Release binary not found. Starting Rock-OS from Go source...
    go run . --host local
)

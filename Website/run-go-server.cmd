@echo off
setlocal

rem Source-only launcher for development or troubleshooting when you want to run
rem the Go server directly instead of using or downloading a release binary.

cd /d "%~dp0"

set "GOCACHE=%CD%\.gocache"

echo Starting Rock-OS from Go source...
go run main.go --host local

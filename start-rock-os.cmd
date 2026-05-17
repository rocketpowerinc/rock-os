@echo off
setlocal

cd /d "%~dp0Website"
go run . --host local

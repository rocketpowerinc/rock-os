@echo off
setlocal

rem Source-only LAN launcher. This intentionally exposes Rock-OS to trusted
rem devices on your local network while still building from local Go source.

call "%~dp0start-rock-os-from-source.cmd" lan

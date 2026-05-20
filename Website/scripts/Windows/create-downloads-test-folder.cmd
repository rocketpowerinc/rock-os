@echo off
setlocal

set "TARGET=%USERPROFILE%\Downloads\Rock-OS-Script-Test"

echo This test script creates this folder:
echo %TARGET%

mkdir "%TARGET%" 2>nul
if errorlevel 1 (
    echo Failed to create the folder.
    exit /b 1
)

echo Folder is ready:
echo %TARGET%

@echo off
setlocal

cd /d "%~dp0"

set "GIT_CRYPT=git-crypt"
where git-crypt >nul 2>nul
if errorlevel 1 (
    if exist "%USERPROFILE%\scoop\shims\git-crypt.exe" (
        set "GIT_CRYPT=%USERPROFILE%\scoop\shims\git-crypt.exe"
    ) else (
    echo git-crypt was not found.
    echo Install git-crypt, then run this script again.
    exit /b 1
    )
)

set "KEY_FILE="
for %%F in (*.key) do (
    if not defined KEY_FILE (
        set "KEY_FILE=%%~fF"
    ) else (
        echo More than one .key file was found in the repo root.
        echo Keep only the git-crypt key here, then run this script again.
        exit /b 1
    )
)

if not defined KEY_FILE (
    echo No .key file was found in the repo root.
    echo Copy your exported git-crypt key here, then run this script again.
    exit /b 1
)

echo Unlocking repository with "%KEY_FILE%"...
"%GIT_CRYPT%" unlock "%KEY_FILE%"
if errorlevel 1 (
    echo Failed to unlock the repository.
    exit /b 1
)

echo Repository unlocked.

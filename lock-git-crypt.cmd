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
        call :wait
        exit /b 1
    )
)

if not exist ".git" (
    echo This script must be run from the Rock-OS repo root.
    call :wait
    exit /b 1
)

echo Locking private markdown with git-crypt...
"%GIT_CRYPT%" lock
set "LOCK_RESULT=%ERRORLEVEL%"

if not "%LOCK_RESULT%"=="0" (
    echo Failed to lock the repository.
    echo Close open private files or commit/stash changes, then try again.
    call :wait
    exit /b %LOCK_RESULT%
)

echo Repository locked.
call :wait

:wait
echo.
pause
exit /b 0

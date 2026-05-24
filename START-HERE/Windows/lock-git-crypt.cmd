@echo off
setlocal

for %%I in ("%~dp0..\..") do set "ROCK_OS_ROOT=%%~fI"
cd /d "%ROCK_OS_ROOT%"

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
    echo This script must be run from inside a cloned Rock-OS repo.
    echo Expected repo root:
    echo "%ROCK_OS_ROOT%"
    call :wait
    exit /b 1
)

echo Locking Profiles with git-crypt...
"%GIT_CRYPT%" lock
set "LOCK_RESULT=%ERRORLEVEL%"

if not "%LOCK_RESULT%"=="0" (
    echo Failed to lock the repository.
    echo Close open Profiles files or commit/stash changes, then try again.
    call :wait
    exit /b %LOCK_RESULT%
)

echo Repository locked.
call :wait
exit /b 0

:wait
echo.
pause
exit /b 0

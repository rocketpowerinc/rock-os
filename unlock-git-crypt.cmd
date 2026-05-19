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

set "KEY_FILE="
set "KEY_NAME="
for %%F in (*.key) do (
    if not defined KEY_FILE (
        set "KEY_FILE=%%~fF"
        set "KEY_NAME=%%~nxF"
    ) else (
        echo More than one .key file was found in the repo root.
        echo Keep only the git-crypt key here, then run this script again.
        call :wait
        exit /b 1
    )
)

if not defined KEY_FILE (
    echo No .key file was found in the repo root.
    echo Copy your exported git-crypt key here, then run this script again.
    call :wait
    exit /b 1
)

echo Unlocking repository with "%KEY_FILE%"...
set "TEMP_KEY=%TEMP%\rock-os-git-crypt-%RANDOM%-%RANDOM%.key"
copy /Y "%KEY_FILE%" "%TEMP_KEY%" >nul
if errorlevel 1 (
    echo Failed to copy the key to a temporary location.
    call :wait
    exit /b 1
)

del "%KEY_FILE%"
if errorlevel 1 (
    echo Failed to remove the temporary root key file.
    echo Remove "%KEY_FILE%" manually, then run this script again.
    del "%TEMP_KEY%" >nul 2>nul
    call :wait
    exit /b 1
)

"%GIT_CRYPT%" unlock "%TEMP_KEY%"
set "UNLOCK_RESULT=%ERRORLEVEL%"

copy /Y "%TEMP_KEY%" "%~dp0%KEY_NAME%" >nul
if errorlevel 1 (
    echo Failed to copy the key back to the repo root.
    echo Your key is still at "%TEMP_KEY%".
    call :wait
    exit /b 1
)

del "%TEMP_KEY%" >nul 2>nul

if not "%UNLOCK_RESULT%"=="0" (
    echo Failed to unlock the repository.
    call :wait
    exit /b %UNLOCK_RESULT%
)

echo Repository unlocked.
echo Key restored to "%~dp0%KEY_NAME%".
call :wait

:wait
echo.
pause
exit /b 0

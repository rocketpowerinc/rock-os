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
    echo Repo root: "%ROCK_OS_ROOT%"
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
        echo More than one .key file was found in the repo root:
        echo "%ROCK_OS_ROOT%"
        echo Keep only the git-crypt key in that folder, then run this script again.
        call :wait
        exit /b 1
    )
)

if not defined KEY_FILE (
    echo No .key file was found in the repo root:
    echo "%ROCK_OS_ROOT%"
    echo Copy your exported git-crypt key to the repo root folder, then run this script again.
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

copy /Y "%TEMP_KEY%" "%ROCK_OS_ROOT%\%KEY_NAME%" >nul
if errorlevel 1 (
    echo Failed to copy the key back to the repo root.
    echo Your key is still at "%TEMP_KEY%".
    call :wait
    exit /b 1
)

del "%TEMP_KEY%" >nul 2>nul

if not "%UNLOCK_RESULT%"=="0" (
    echo Failed to unlock the repository.
    echo If you see "unable to write key file", check permissions on .git\git-crypt\keys.
    call :wait
    exit /b %UNLOCK_RESULT%
)

call :verify_rocket_unlocked
if errorlevel 1 (
    call :wait
    exit /b 1
)

echo Repository unlocked.
echo Key restored to "%ROCK_OS_ROOT%\%KEY_NAME%".
call :wait
exit /b 0

:verify_rocket_unlocked
powershell -NoProfile -ExecutionPolicy Bypass -Command "$files = git ls-files -- 'Website/menu/rocket' 2>$null; foreach ($file in $files) { if (-not (Test-Path -LiteralPath $file)) { continue }; $bytes = [IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $file)); if ($bytes.Length -ge 10 -and [Text.Encoding]::ASCII.GetString($bytes, 1, 8) -eq 'GITCRYPT') { exit 2 } }; exit 0"
set "VERIFY_RESULT=%ERRORLEVEL%"
if "%VERIFY_RESULT%"=="0" (
    echo Rocket markdown verified unlocked.
    exit /b 0
)
if not "%VERIFY_RESULT%"=="2" (
    echo Could not verify Rocket markdown unlock state.
    exit /b 1
)

echo Rocket files still look encrypted. Refreshing clean Rocket files...
set "ROCKET_DIRTY="
for /f "delims=" %%S in ('git status --porcelain -- "Website/menu/rocket" 2^>nul') do set "ROCKET_DIRTY=1"
if defined ROCKET_DIRTY (
    echo Rocket markdown has local changes, so this script will not restore it automatically.
    echo Back up or clear those changes first, then run:
    echo git restore --source=HEAD --worktree -- Website/menu/rocket
    exit /b 1
)

for /f "delims=" %%F in ('git ls-files -- "Website/menu/rocket" 2^>nul') do (
    if exist "%%F" del /f /q "%%F" >nul 2>nul
)

git restore --source=HEAD --worktree -- "Website/menu/rocket" >nul 2>nul
if errorlevel 1 git checkout -- "Website/menu/rocket" >nul 2>nul

powershell -NoProfile -ExecutionPolicy Bypass -Command "$files = git ls-files -- 'Website/menu/rocket' 2>$null; foreach ($file in $files) { if (-not (Test-Path -LiteralPath $file)) { continue }; $bytes = [IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $file)); if ($bytes.Length -ge 10 -and [Text.Encoding]::ASCII.GetString($bytes, 1, 8) -eq 'GITCRYPT') { exit 2 } }; exit 0"
set "VERIFY_RESULT=%ERRORLEVEL%"
if "%VERIFY_RESULT%"=="2" (
    echo Rocket markdown still looks encrypted after refresh.
    exit /b 1
)
if not "%VERIFY_RESULT%"=="0" (
    echo Could not verify Rocket markdown unlock state after refresh.
    exit /b 1
)

echo Rocket markdown verified unlocked.
exit /b 0

:wait
echo.
pause
exit /b 0

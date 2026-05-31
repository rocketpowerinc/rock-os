@echo off
setlocal EnableExtensions

goto :main

:green
set "ROCK_OS_MSG=%~1"
powershell -NoProfile -Command "Write-Host $env:ROCK_OS_MSG -ForegroundColor Green" 2>nul
if errorlevel 1 echo %~1
exit /b 0

:yellow
set "ROCK_OS_MSG=%~1"
powershell -NoProfile -Command "Write-Host $env:ROCK_OS_MSG -ForegroundColor Yellow" 2>nul
if errorlevel 1 echo %~1
exit /b 0

:red
set "ROCK_OS_MSG=%~1"
powershell -NoProfile -Command "Write-Host $env:ROCK_OS_MSG -ForegroundColor Red" 2>nul
if errorlevel 1 echo %~1
exit /b 0

:pull_updates
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command git -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
if errorlevel 1 (
    call :yellow "Git is not installed. Skipping repo update and using local files."
    call :yellow "To enable auto-updates, install Git with:"
    echo.
    echo   winget.exe install --id "Git.Git" --exact --source winget --accept-source-agreements --disable-interactivity --silent --accept-package-agreements --force
    echo.
    exit /b 0
)
call :green "Checking for Rock-OS repo updates..."
set "ROCK_OS_PULL_BEFORE="
for /f "delims=" %%H in ('git -C "%ROCK_OS_ROOT%" rev-parse HEAD 2^>nul') do set "ROCK_OS_PULL_BEFORE=%%H"
git -C "%ROCK_OS_ROOT%" pull --ff-only
if errorlevel 1 (
    call :yellow "Could not update from GitHub. Continuing with local files."
    call :yellow "If you have local changes, commit them before pulling updates."
    exit /b 0
)
set "ROCK_OS_PULL_AFTER="
for /f "delims=" %%H in ('git -C "%ROCK_OS_ROOT%" rev-parse HEAD 2^>nul') do set "ROCK_OS_PULL_AFTER=%%H"
call :green "Rock-OS repo is up to date."
if defined ROCK_OS_PULL_BEFORE if defined ROCK_OS_PULL_AFTER if /I not "%ROCK_OS_PULL_BEFORE%"=="%ROCK_OS_PULL_AFTER%" (
    if not "%ROCK_OS_RESTARTED_AFTER_PULL%"=="1" (
        exit /b 222
    )
)
exit /b 0

:check_release_binary
powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $repo=$env:ROCK_OS_REPO; $stableAsset=$env:ROCK_OS_STABLE_ASSET; $versionFile=$env:ROCK_OS_VERSION_FILE; $arch=$env:ROCK_OS_ARCH; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $release=Invoke-RestMethod -Uri ('https://api.github.com/repos/{0}/releases/latest' -f $repo) -Headers @{'User-Agent'='rock-os-start-script'}; $tag=$release.tag_name; $versionedAsset=('rock-os-{0}-windows-{1}.exe' -f $tag,$arch); $local=''; if (Test-Path $versionFile) { $lines=@(Get-Content $versionFile | ForEach-Object { $line=$_.Trim(); if ($line -and -not $line.StartsWith('#')) { $line } }); if ($lines.Count -gt 0) { $local=$lines[0] } }; if ((-not (Test-Path $stableAsset)) -or ($local -ne $tag)) { Write-Host ('Downloading Rock-OS {0} for Windows {1}...' -f $tag,$arch) -ForegroundColor Yellow; $downloaded=$false; foreach ($asset in @($stableAsset,$versionedAsset)) { $tempFile=('{0}.download' -f $asset); if (Test-Path $tempFile) { Remove-Item $tempFile -Force }; try { Invoke-WebRequest -Uri ('https://github.com/{0}/releases/latest/download/{1}' -f $repo,$asset) -OutFile $tempFile -Headers @{'User-Agent'='rock-os-start-script'}; if ((Test-Path $tempFile) -and ((Get-Item $tempFile).Length -gt 0)) { Move-Item $tempFile $stableAsset -Force; Set-Content -Path $versionFile -Value @($tag,'# Local Rock-OS release marker used by START-HERE launchers.','# First non-comment line is the downloaded release tag.','# If this tag differs from GitHub''s latest release, the launcher downloads a fresh binary.'); Write-Host ('Downloaded Rock-OS {0}.' -f $tag) -ForegroundColor Green; $downloaded=$true; break } } catch { if (Test-Path $tempFile) { Remove-Item $tempFile -Force } } }; if (-not $downloaded) { exit 1 } } else { Write-Host ('Rock-OS binary is current ({0}).' -f $tag) -ForegroundColor Green }" 2>nul
exit /b %ERRORLEVEL%

:check_go
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command go -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
if errorlevel 1 (
    call :yellow "Go is not installed. Not needed while using a release binary."
    exit /b 0
)
call :green "Go installed. Source fallback available."
exit /b 0

:wait
echo.
pause
exit /b 0

:main
for %%I in ("%~dp0..\..") do set "ROCK_OS_ROOT=%%~fI"

set "ROCK_OS_EXIT=0"
set "ROCK_OS_HOST=127.0.0.1"
if /I "%~1"=="lan" set "ROCK_OS_HOST=local"
if /I "%~1"=="local" set "ROCK_OS_HOST=local"
if /I "%~1"=="all" set "ROCK_OS_HOST=local"
if /I "%~1"=="0.0.0.0" set "ROCK_OS_HOST=0.0.0.0"
if /I "%~1"=="127.0.0.1" set "ROCK_OS_HOST=127.0.0.1"

if exist "%ROCK_OS_ROOT%\.git" (
    set "ROCK_OS_HAS_GIT=1"
) else (
    set "ROCK_OS_HAS_GIT=0"
    call :red    "============================================================"
    call :red    "  WARNING: This is NOT a cloned Git repository."
    call :red    "============================================================"
    call :yellow "  Rock-OS will still start, but in a LIMITED mode:"
    call :yellow "    - Automatic updates are skipped (no 'git pull')."
    call :yellow "    - git-crypt cannot unlock private Profiles."
    echo.
    call :yellow "  This usually means Rock-OS was downloaded as a GitHub ZIP,"
    call :yellow "  which does not include the hidden .git folder."
    echo.
    call :yellow "  For updates and Profiles, a real clone is strongly recommended:"
    call :green  "    git clone https://github.com/rocketpowerinc/rock-os.git"
    call :green  "    cd rock-os\START-HERE\Windows"
    call :red    "============================================================"
    call :yellow "  Press Enter to continue from local files, or close this window to cancel."
    set /p "ROCK_OS_CONTINUE=  > "
)

if "%ROCK_OS_HAS_GIT%"=="0" goto :after_pull
call :pull_updates
if "%ERRORLEVEL%"=="222" (
    call :yellow "Launcher files changed during update. Restarting Rock-OS launcher once..."
    set "ROCK_OS_RESTARTED_AFTER_PULL=1"
    call "%~f0" %*
    exit /b %ERRORLEVEL%
)
:after_pull

cd /d "%ROCK_OS_ROOT%\Website"
if errorlevel 1 (
    call :red "Could not enter the Website folder."
    call :wait
    exit /b 1
)

set "ROCK_OS_REPO=rocketpowerinc/rock-os"
set "ROCK_OS_VERSION_FILE=.rock-os-version"
set "ROCK_OS_BINARY="

set "ROCK_OS_ARCH=%PROCESSOR_ARCHITEW6432%"
if "%ROCK_OS_ARCH%"=="" set "ROCK_OS_ARCH=%PROCESSOR_ARCHITECTURE%"
if /I "%ROCK_OS_ARCH%"=="ARM64" (
    set "ROCK_OS_ARCH=arm64"
) else (
    set "ROCK_OS_ARCH=amd64"
)

set "ROCK_OS_STABLE_ASSET=rock-os-windows-%ROCK_OS_ARCH%.exe"

call :check_go
call :check_release_binary
if errorlevel 1 (
    call :yellow "Could not check or download the latest Rock-OS binary. Continuing with local files..."
)

if exist ".\%ROCK_OS_STABLE_ASSET%" (
    set "ROCK_OS_BINARY=.\%ROCK_OS_STABLE_ASSET%"
) else (
    for /f "delims=" %%F in ('dir /b /o:-n ".\rock-os-v*-windows-%ROCK_OS_ARCH%.exe" 2^>nul') do (
        if not defined ROCK_OS_BINARY (
            set "ROCK_OS_BINARY=.\%%F"
        )
    )
)

if defined ROCK_OS_BINARY (
    call :green "Starting Rock-OS..."
    "%ROCK_OS_BINARY%" --host "%ROCK_OS_HOST%"
    set "ROCK_OS_EXIT=%ERRORLEVEL%"
) else (
    call :yellow "Release binary not found. Starting Rock-OS from Go source..."
    powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command go -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
    if errorlevel 1 (
        call :red "Cannot start from source because Go is not installed."
        call :wait
        exit /b 1
    )
    set "GOCACHE=%CD%\.gocache"
    set "GOTMPDIR=%CD%\.gotmp"
    if not exist "%GOCACHE%" mkdir "%GOCACHE%" >nul 2>nul
    if not exist "%GOTMPDIR%" mkdir "%GOTMPDIR%" >nul 2>nul
    set "ROCK_OS_WEBSITE=%CD%"
    pushd "%ROCK_OS_ROOT%\cmd\rock-os"
    go run . --site-root "%ROCK_OS_WEBSITE%" --host "%ROCK_OS_HOST%"
    set "ROCK_OS_EXIT=%ERRORLEVEL%"
    popd
)

call :wait
exit /b %ROCK_OS_EXIT%

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
    exit /b 0
)
call :green "Checking for Rock-OS repo updates..."
git -C "%ROCK_OS_ROOT%" pull --ff-only
if errorlevel 1 (
    call :yellow "Could not update from GitHub. Continuing with local files."
    call :yellow "If you have local changes, commit them before pulling updates."
    exit /b 0
)
call :green "Rock-OS repo is up to date."
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

if not exist "%ROCK_OS_ROOT%\.git" (
    call :red "This folder is not a cloned Git repository."
    call :yellow "GitHub ZIP downloads do not include the .git folder, so git-crypt cannot unlock Profiles."
    call :yellow "Use this instead:"
    echo git clone https://github.com/rocketpowerinc/rock-os.git
    echo cd rock-os
    echo cd START-HERE\Windows
    call :wait
    exit /b 1
)

call :pull_updates

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
    set "ROCK_OS_WEBSITE=%CD%"
    pushd "%ROCK_OS_ROOT%\cmd\rock-os"
    go run . --site-root "%ROCK_OS_WEBSITE%" --host "%ROCK_OS_HOST%"
    set "ROCK_OS_EXIT=%ERRORLEVEL%"
    popd
)

call :wait
exit /b %ROCK_OS_EXIT%

@echo off
setlocal

cd /d "%~dp0Website"

set "ROCK_OS_REPO=rocketpowerinc/rock-os"
set "ROCK_OS_VERSION_FILE=.rock-os-wiki-version"
set "ROCK_OS_BINARY="
set "ROCK_OS_BINARY_SOURCE="

call :green "[Rock-OS] Launcher online."

set "ROCK_OS_ARCH=%PROCESSOR_ARCHITEW6432%"
if "%ROCK_OS_ARCH%"=="" set "ROCK_OS_ARCH=%PROCESSOR_ARCHITECTURE%"
if /I "%ROCK_OS_ARCH%"=="ARM64" (
    set "ROCK_OS_ARCH=arm64"
) else (
    set "ROCK_OS_ARCH=amd64"
)

set "ROCK_OS_STABLE_ASSET=rock-os-wiki-windows-%ROCK_OS_ARCH%.exe"
call :green "Detected Windows %ROCK_OS_ARCH%."

powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $repo=$env:ROCK_OS_REPO; $stableAsset=$env:ROCK_OS_STABLE_ASSET; $versionFile=$env:ROCK_OS_VERSION_FILE; $arch=$env:ROCK_OS_ARCH; [Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12; $release=Invoke-RestMethod -Uri ('https://api.github.com/repos/{0}/releases/latest' -f $repo) -Headers @{'User-Agent'='rock-os-start-script'}; $tag=$release.tag_name; $versionedAsset=('rock-os-wiki-{0}-windows-{1}.exe' -f $tag,$arch); $local=''; if (Test-Path $versionFile) { $local=(Get-Content $versionFile -Raw).Trim() }; if ((-not (Test-Path $stableAsset)) -or ($local -ne $tag)) { Write-Host ('Downloading Rock-OS {0} for Windows {1}...' -f $tag,$arch) -ForegroundColor Yellow; $downloaded=$false; foreach ($asset in @($stableAsset,$versionedAsset)) { $tempFile=('{0}.download' -f $asset); if (Test-Path $tempFile) { Remove-Item $tempFile -Force }; try { Invoke-WebRequest -Uri ('https://github.com/{0}/releases/latest/download/{1}' -f $repo,$asset) -OutFile $tempFile -Headers @{'User-Agent'='rock-os-start-script'}; if ((Test-Path $tempFile) -and ((Get-Item $tempFile).Length -gt 0)) { Move-Item $tempFile $stableAsset -Force; Set-Content -Path $versionFile -Value $tag -NoNewline; Write-Host ('Downloaded Rock-OS {0}.' -f $tag) -ForegroundColor Green; $downloaded=$true; break } } catch { if (Test-Path $tempFile) { Remove-Item $tempFile -Force } } }; if (-not $downloaded) { exit 1 } } else { Write-Host ('Rock-OS binary is current ({0}).' -f $tag) -ForegroundColor Green }" 2>nul
if errorlevel 1 (
    call :yellow "Could not check or download the latest Rock-OS binary. Continuing with local files..."
)

if exist ".\%ROCK_OS_STABLE_ASSET%" (
    set "ROCK_OS_BINARY=.\%ROCK_OS_STABLE_ASSET%"
    set "ROCK_OS_BINARY_SOURCE=stable"
) else (
    for /f "delims=" %%F in ('dir /b /o:-n ".\rock-os-wiki-v*-windows-%ROCK_OS_ARCH%.exe" 2^>nul') do (
        if not defined ROCK_OS_BINARY (
            set "ROCK_OS_BINARY=.\%%F"
            set "ROCK_OS_BINARY_SOURCE=versioned"
        )
    )
)

call :check_git_crypt
call :check_go
call :check_private

if defined ROCK_OS_BINARY (
    if "%ROCK_OS_BINARY_SOURCE%"=="stable" (
        call :green "Release binary found: %ROCK_OS_BINARY%"
    ) else (
        call :yellow "Using versioned fallback binary: %ROCK_OS_BINARY%"
    )
    call :green "Starting Rock-OS..."
    "%ROCK_OS_BINARY%" --host local
) else (
    call :yellow "Release binary not found. Starting Rock-OS from Go source..."
    powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command go -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
    if errorlevel 1 (
        call :red "Cannot start from source because Go is not installed."
        exit /b 1
    )
    set "GOCACHE=%CD%\.gocache"
    go run . --host local
)

exit /b %ERRORLEVEL%

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

:check_git_crypt
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command git-crypt -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
if errorlevel 1 (
    call :red "git-crypt is not installed. Install git-crypt before unlocking Private markdown."
    exit /b 0
)
call :green "git-crypt is installed."
exit /b 0

:check_go
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command go -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" 2>nul
if errorlevel 1 (
    if defined ROCK_OS_BINARY (
        call :yellow "Go is not installed. Not needed while using a release binary."
    ) else (
        call :red "Go is not installed. Install Go from https://go.dev/dl/ before using source fallback."
    )
    exit /b 0
)
call :green "Go is installed."
exit /b 0

:check_private
powershell -NoProfile -ExecutionPolicy Bypass -Command "$files=git -C .. ls-files -- 'Website/markdown/Private' 2>$null; if (-not $files) { exit 0 }; foreach ($file in $files) { $path=Join-Path '..' $file; if (Test-Path $path) { $bytes=[IO.File]::ReadAllBytes((Resolve-Path $path)); if ($bytes.Length -ge 10 -and [Text.Encoding]::ASCII.GetString($bytes,1,8) -eq 'GITCRYPT') { exit 2 } } }; exit 0" 2>nul
if errorlevel 2 (
    call :red "Private Markdown Folder Locked."
    exit /b 0
)
if errorlevel 1 (
    call :yellow "Could not verify private markdown unlock status."
    exit /b 0
)
call :green "Private Markdown Folder Unlocked."
exit /b 0

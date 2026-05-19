@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

echo.
call :green "== Rock-OS Repo Status =="

if not exist ".git" (
    call :red "This folder is not a cloned Git repository."
    call :yellow "Use: git clone https://github.com/rocketpowerinc/rock-os.git"
    call :wait
    exit /b 1
)

call :section "Git"
for /f "delims=" %%V in ('git --version 2^>nul') do call :ok "%%V"

for /f "delims=" %%B in ('git branch --show-current 2^>nul') do set "BRANCH=%%B"
if not defined BRANCH set "BRANCH=(detached HEAD)"
call :info "Branch: !BRANCH!"

for /f "delims=" %%H in ('git rev-parse --short HEAD 2^>nul') do call :info "Commit: %%H"
for /f "delims=" %%C in ('git rev-list --count HEAD 2^>nul') do call :info "Total commits: %%C"
for /f "delims=" %%U in ('git rev-parse --abbrev-ref --symbolic-full-name @{u} 2^>nul') do set "UPSTREAM=%%U"
if defined UPSTREAM (
    call :info "Upstream: !UPSTREAM!"
) else (
    call :warn "No upstream branch configured."
)

for /f "delims=" %%S in ('git status -sb 2^>nul') do (
    set "STATUS_LINE=%%S"
    goto :printed_status_line
)
:printed_status_line
if defined STATUS_LINE call :info "!STATUS_LINE!"

set "DIRTY="
for /f "delims=" %%S in ('git status --short 2^>nul') do (
    if not defined DIRTY (
        call :warn "Working tree has changes:"
        set "DIRTY=1"
    )
    echo   %%S
)
if not defined DIRTY call :ok "Working tree clean."

call :section "Website"
if exist "Website\main.go" (
    call :ok "Go server source present for source fallback."
) else (
    call :bad "Website\main.go missing."
)

set "BINARY_FOUND="
for %%B in (Website\rock-os-wiki-*) do set "BINARY_FOUND=1"
if defined BINARY_FOUND (
    call :ok "Release binary present. Site can run without Go installed."
) else (
    call :warn "No release binary found in Website folder."
)

call :section "Tools"
where go >nul 2>nul
if errorlevel 1 (
    call :warn "Go is not installed or not on PATH. Not needed if using release binary."
) else (
    for /f "delims=" %%G in ('go version 2^>nul') do call :ok "%%G"
)

call :section "Port 8000"
set "PORT_OPEN="
for /f "tokens=5" %%P in ('netstat -ano ^| findstr /R /C:":8000 .*LISTENING" 2^>nul') do (
    set "PORT_OPEN=1"
    call :ok "Port 8000 is listening on PID %%P."
)
if not defined PORT_OPEN call :info "Port 8000 is not currently listening."

call :section "git-crypt"
where git-crypt >nul 2>nul
if errorlevel 1 (
    if exist "%USERPROFILE%\scoop\shims\git-crypt.exe" (
        set "GIT_CRYPT=%USERPROFILE%\scoop\shims\git-crypt.exe"
        call :ok "git-crypt installed via Scoop."
    ) else (
        call :bad "git-crypt is not installed."
        set "GIT_CRYPT="
    )
) else (
    set "GIT_CRYPT=git-crypt"
    call :ok "git-crypt installed."
)

if defined GIT_CRYPT (
    "%GIT_CRYPT%" status >nul 2>nul
    if errorlevel 1 (
        call :warn "git-crypt status could not be read."
    ) else (
        call :ok "git-crypt status is available."
    )
)

call :check_private

set "KEY_FOUND="
for %%K in (*.key) do (
    set "KEY_FOUND=1"
)
if defined KEY_FOUND (
    call :warn ".key file present in repo root. Keep it private and never commit it."
) else (
    call :ok "No .key files found in repo root."
)

call :section "Full git-crypt status"
if defined GIT_CRYPT (
    "%GIT_CRYPT%" status
    if errorlevel 1 call :warn "git-crypt status exited with an error."
) else (
    call :bad "git-crypt is not installed."
)

call :section "Done"
call :ok "Repo status check complete."
call :wait
exit /b 0

:check_private
powershell -NoProfile -ExecutionPolicy Bypass -Command "$files=git ls-files -- 'Website/markdown/Private' 2>$null; if (-not $files) { exit 3 }; foreach ($file in $files) { if (Test-Path $file) { $bytes=[IO.File]::ReadAllBytes((Resolve-Path $file)); if ($bytes.Length -ge 10 -and [Text.Encoding]::ASCII.GetString($bytes,1,8) -eq 'GITCRYPT') { exit 2 } } }; exit 0" 2>nul
if errorlevel 3 (
    call :info "No tracked Private markdown files found."
    exit /b 0
)
if errorlevel 2 (
    call :bad "Private Markdown Folder Locked."
    exit /b 0
)
if errorlevel 1 (
    call :warn "Could not verify private markdown status."
    exit /b 0
)
call :ok "Private Markdown Folder Unlocked."
exit /b 0

:section
echo.
call :green "-- %~1 --"
exit /b 0

:ok
call :green "[OK] %~1"
exit /b 0

:info
call :green "[INFO] %~1"
exit /b 0

:warn
call :yellow "[WARN] %~1"
exit /b 0

:bad
call :red "[BAD] %~1"
exit /b 0

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

:wait
echo.
pause
exit /b 0

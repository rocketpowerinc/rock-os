$ErrorActionPreference = 'Stop'

Write-Host 'This script updates installed Windows packages through winget.'
Write-Host 'It accepts source and package agreements so it can run unattended.'
Write-Host ''

if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Host 'winget was not found. Install App Installer from Microsoft Store, then try again.'
    exit 1
}

winget upgrade --all --include-unknown --accept-source-agreements --accept-package-agreements

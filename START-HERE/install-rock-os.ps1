$ErrorActionPreference = 'Stop'

$LocalInstaller = Join-Path $PSScriptRoot 'Windows\install-rock-os.ps1'
if ($PSScriptRoot -and (Test-Path $LocalInstaller)) {
    & $LocalInstaller @args
    exit $LASTEXITCODE
}

irm https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Windows/install-rock-os.ps1 | iex

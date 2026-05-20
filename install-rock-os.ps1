$ErrorActionPreference = 'Stop'

$RepoUrl = 'https://github.com/rocketpowerinc/rock-os.git'
$InstallDir = Join-Path $HOME 'rock-os'
$BinDir = Join-Path $HOME 'Bin'
$RockCommand = Join-Path $BinDir 'rock-os.cmd'
$LegacyRockCommand = Join-Path $BinDir 'rock.cmd'
$DesktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'Rock-OS.lnk'

function Write-Green($Message) {
    Write-Host $Message -ForegroundColor Green
}

function Write-Yellow($Message) {
    Write-Host $Message -ForegroundColor Yellow
}

function Ensure-Git {
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        throw 'Git is required. Install Git, then run this installer again.'
    }
}

function Ensure-Repo {
    if (Test-Path (Join-Path $InstallDir '.git')) {
        Write-Green "Rock-OS repo found at $InstallDir"
        git -C $InstallDir pull --ff-only
        return
    }

    if (Test-Path $InstallDir) {
        throw "$InstallDir exists but is not a Git clone. Move it or remove it, then run this installer again."
    }

    Write-Green "Cloning Rock-OS into $InstallDir"
    git clone $RepoUrl $InstallDir
}

function Ensure-BinOnPath {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null

    $userPath =
        [Environment]::GetEnvironmentVariable('Path', 'User')

    $pathParts =
        ($userPath -split ';') | Where-Object { $_ }

    if ($pathParts -notcontains $BinDir) {
        $nextPath =
            (@($pathParts) + $BinDir) -join ';'

        [Environment]::SetEnvironmentVariable('Path', $nextPath, 'User')
        $env:Path = "$env:Path;$BinDir"
        Write-Yellow "Added $BinDir to your user PATH. New terminals can run rock-os."
    }
}

function Write-RockCommand {
    $startScript =
        Join-Path $InstallDir 'start-rock-os.cmd'

    if (Test-Path $LegacyRockCommand) {
        Remove-Item -Path $LegacyRockCommand -Force
        Write-Yellow 'Removed old terminal command: rock'
    }

    @"
@echo off
call "$startScript" %*
"@ | Set-Content -Path $RockCommand -Encoding ASCII

    Write-Green "Created terminal command: rock-os"
}

function Create-DesktopShortcut {
    $startScript =
        Join-Path $InstallDir 'start-rock-os.cmd'
    $iconPath =
        Join-Path $InstallDir 'Website\assets\favicon.ico'

    $shell =
        New-Object -ComObject WScript.Shell
    $shortcut =
        $shell.CreateShortcut($DesktopShortcut)

    $shortcut.TargetPath = $startScript
    $shortcut.WorkingDirectory = $InstallDir
    $shortcut.IconLocation = $iconPath
    $shortcut.Description = 'Start Rock-OS'
    $shortcut.Save()

    Write-Green "Created desktop shortcut: $DesktopShortcut"
}

Ensure-Git
Ensure-Repo
Ensure-BinOnPath
Write-RockCommand
Create-DesktopShortcut

Write-Green ''
Write-Green 'Rock-OS is installed.'
Write-Green 'Run it from a new terminal with: rock-os'
Write-Green 'Or use the Rock-OS desktop shortcut.'
Write-Green 'Starting Rock-OS now...'

& (Join-Path $InstallDir 'start-rock-os.cmd')

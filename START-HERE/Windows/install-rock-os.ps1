$ErrorActionPreference = 'Stop'

$RepoUrl = 'https://github.com/rocketpowerinc/rock-os.git'
$UserRoot = $env:USERPROFILE
$InstallDir = Join-Path $UserRoot 'rock-os'
$BinDir = Join-Path $UserRoot 'Bin'
$RockCommand = Join-Path $BinDir 'rock-os.cmd'
$DesktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'Rock-OS.lnk'

function Write-Green($Message) {
    Write-Host $Message -ForegroundColor Green
}

function Write-Yellow($Message) {
    Write-Host $Message -ForegroundColor Yellow
}

function Ensure-Git {
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Yellow 'Git is required but was not found on your PATH.'
        Write-Yellow 'Install Git with this command, then run this installer again:'
        Write-Host ''
        Write-Host '  winget.exe install --id "Git.Git" --exact --source winget --accept-source-agreements --disable-interactivity --silent --accept-package-agreements --force'
        Write-Host ''
        throw 'Git is required. Run the install command shown above, then re-run this installer.'
    }
}

function Ensure-Repo {
    if (Test-Path (Join-Path $InstallDir '.git')) {
        Write-Green "Rock-OS repo found at $InstallDir"
        git -C $InstallDir pull --ff-only
        if ($LASTEXITCODE -eq 0) {
            Write-Green "Rock-OS repo is up to date."
        }
        else {
            Write-Yellow "Could not update the repo. Continuing so launcher files can still be refreshed."
            Write-Yellow "If you have local changes, commit them before pulling updates."
        }
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
        Join-Path $InstallDir 'START-HERE\Windows\start-rock-os.cmd'

    @"
@echo off
call "$startScript" %*
"@ | Set-Content -Path $RockCommand -Encoding ASCII

    Write-Green "Created terminal command: rock-os"
}

function Create-DesktopShortcut {
    $startScript =
        Join-Path $InstallDir 'START-HERE\Windows\start-rock-os.cmd'
    $iconPath =
        Join-Path $InstallDir 'Website\assets\favicon.ico'

    if (Test-Path $DesktopShortcut) {
        Remove-Item -Path $DesktopShortcut -Force
    }

    $shell =
        New-Object -ComObject WScript.Shell
    $shortcut =
        $shell.CreateShortcut($DesktopShortcut)

    $shortcut.TargetPath = $startScript
    $shortcut.WorkingDirectory = Join-Path $InstallDir 'START-HERE\Windows'
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

& (Join-Path $InstallDir 'START-HERE\Windows\start-rock-os.cmd')

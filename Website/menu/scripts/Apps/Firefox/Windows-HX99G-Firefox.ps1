# HX99G-Firefox.ps1
# Installs Firefox via winget if missing, then configures it with a Rock-OS
# enterprise policy: uBlock Origin, always-visible bookmarks toolbar, and
# a set of toolbar bookmarks. Close Firefox before running this script so
# policies reload cleanly.
#
# Firefox reads enterprise policies from distribution\policies.json at startup.
# This script merges a small Rock-OS policy into that file instead of editing
# Firefox's profile database directly, which is safer and easier to review.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# ── Self-elevate to Administrator ────────────────────────────────────────────
# Writing to Program Files\Mozilla Firefox\distribution requires elevation.

$isAdmin = ([Security.Principal.WindowsPrincipal] `
    [Security.Principal.WindowsIdentity]::GetCurrent()
).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host 'Requesting Administrator privileges...'
    $argList = "-ExecutionPolicy Bypass -File `"$PSCommandPath`""
    Start-Process powershell.exe -Verb RunAs -ArgumentList $argList
    exit
}

Write-Host ''
Write-Host 'This script installs and configures Firefox with a Rock-OS policy.'
Write-Host 'It will:'
Write-Host '  - Install Firefox via winget if not already installed'
Write-Host '  - Always show the bookmarks toolbar'
Write-Host '  - ERASE ALL existing bookmarks from every Firefox profile'
Write-Host '  - Add a bookmark folder to the toolbar'
Write-Host '  - Install uBlock Origin from Mozilla Add-ons'
Write-Host ''
Write-Host '========================================================================' -ForegroundColor Red
Write-Host '  WARNING: This will DELETE every bookmark in every Firefox profile.'   -ForegroundColor Red
Write-Host '  A backup of each places.sqlite is saved before wiping.'               -ForegroundColor Red
Write-Host '========================================================================' -ForegroundColor Red
Write-Host ''
Write-Host 'Close Firefox before running this script so policies reload cleanly.'
Write-Host ''

$yes = New-Object System.Management.Automation.Host.ChoiceDescription '&Yes', 'Erase all bookmarks and apply the policy.'
$no  = New-Object System.Management.Automation.Host.ChoiceDescription '&No',  'Abort without changing anything.'
$choice = $Host.UI.PromptForChoice('Confirm', 'Continue?', @($yes, $no), 1)
if ($choice -ne 0) {
    Write-Host 'Aborted.'
    Read-Host 'Press Enter to exit'
    exit 0
}

Write-Host ''

try {

# ── Install Firefox via winget if not present ────────────────────────────────

$firefoxPaths = @(
    "$env:ProgramFiles\Mozilla Firefox\firefox.exe",
    "${env:ProgramFiles(x86)}\Mozilla Firefox\firefox.exe",
    "$env:LOCALAPPDATA\Mozilla Firefox\firefox.exe"
)

$firefoxExe = $firefoxPaths | Where-Object { Test-Path $_ } | Select-Object -First 1

if (-not $firefoxExe) {
    Write-Host 'Firefox not found. Installing via winget...'
    winget.exe install --id "Mozilla.Firefox" --exact --source winget `
        --accept-source-agreements --disable-interactivity --silent `
        --accept-package-agreements --force

    if ($LASTEXITCODE -ne 0) {
        Write-Host 'winget install failed. Install Firefox manually, then run this script again.'
        Read-Host 'Press Enter to exit'
        exit 1
    }

    # Refresh the search after install
    $firefoxExe = $firefoxPaths | Where-Object { Test-Path $_ } | Select-Object -First 1

    if (-not $firefoxExe) {
        Write-Host 'Firefox was installed but could not be found at the expected paths.'
        Write-Host 'You may need to restart your terminal or check the install location.'
        Read-Host 'Press Enter to exit'
        exit 1
    }

    Write-Host "Firefox installed: $firefoxExe"
} else {
    Write-Host "Firefox already installed: $firefoxExe"
}

# ── Locate or create the distribution policy folder ──────────────────────────

$firefoxDir = Split-Path $firefoxExe -Parent
$policyDir  = Join-Path $firefoxDir 'distribution'
$policyFile = Join-Path $policyDir 'policies.json'

if (-not (Test-Path $policyDir)) {
    New-Item -ItemType Directory -Path $policyDir -Force | Out-Null
    Write-Host "Created policy directory: $policyDir"
}

# ── Load existing policy or start fresh ──────────────────────────────────────

if (Test-Path $policyFile) {
    $timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $backup = "$policyFile.rock-os-backup.$timestamp"
    Copy-Item $policyFile $backup
    Write-Host "Backed up existing policy to $backup"

    try {
        $data = Get-Content $policyFile -Raw -Encoding UTF8 | ConvertFrom-Json
    } catch {
        Write-Host 'Existing policies.json was invalid, starting fresh.'
        $data = [pscustomobject]@{}
    }
} else {
    $data = [pscustomobject]@{}
}

# ── Build the policies object ────────────────────────────────────────────────

if (-not (Get-Member -InputObject $data -Name 'policies' -MemberType NoteProperty)) {
    $data | Add-Member -NotePropertyName 'policies' -NotePropertyValue ([pscustomobject]@{})
}

$policies = $data.policies

# Always show the bookmarks toolbar and suppress the default import prompt
$fieldDefaults = @{
    DisplayBookmarksToolbar = $true
    NoDefaultBookmarks      = $true
    DisableProfileImport    = $true
}

foreach ($key in $fieldDefaults.Keys) {
    if (Get-Member -InputObject $policies -Name $key -MemberType NoteProperty) {
        $policies.$key = $fieldDefaults[$key]
    } else {
        $policies | Add-Member -NotePropertyName $key -NotePropertyValue $fieldDefaults[$key]
    }
}

# ── Bookmarks ────────────────────────────────────────────────────────────────
# All links go inside a toolbar folder. Firefox enterprise policy supports
# Folder + Children to create a named bookmark folder on the toolbar.

$pirateFolder = [pscustomobject]@{
    Title     = [char]0x2B07 + [char]0xFE0F + 'Pirate'   # ⬇️Pirate
    Placement = 'toolbar'
    Folder    = [char]0x2B07 + [char]0xFE0F + 'Pirate'
    Children  = @(
        [pscustomobject]@{ Title = 'SkipVids';       URL = 'https://skipvids.com/' },
        [pscustomobject]@{ Title = 'Ext';            URL = 'https://ext.to/' },
        [pscustomobject]@{ Title = 'TorrentGalaxy';  URL = 'https://torrentgalaxy.one/' },
        [pscustomobject]@{ Title = 'PCGamesTorrent'; URL = 'https://pcgamestorrents.com/' },
        [pscustomobject]@{ Title = 'Ziperto';        URL = 'https://www.ziperto.com/' },
        [pscustomobject]@{ Title = 'DLPSGame';       URL = 'https://dlpsgame.com/category/ps4/' },
        [pscustomobject]@{ Title = 'GetComics';      URL = 'https://getcomics.org/' },
        [pscustomobject]@{ Title = 'PirateBay';      URL = 'https://thepiratebay10.xyz/' },
        [pscustomobject]@{ Title = 'VibeMax';        URL = 'https://vibemax.to/' }
    )
}

$bookmarks = @($pirateFolder)

if (Get-Member -InputObject $policies -Name 'Bookmarks' -MemberType NoteProperty) {
    $policies.Bookmarks = $bookmarks
} else {
    $policies | Add-Member -NotePropertyName 'Bookmarks' -NotePropertyValue $bookmarks
}

# ── uBlock Origin ────────────────────────────────────────────────────────────

$ublock = [pscustomobject]@{
    installation_mode = 'force_installed'
    install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi'
}

if (Get-Member -InputObject $policies -Name 'ExtensionSettings' -MemberType NoteProperty) {
    $ext = $policies.ExtensionSettings
    if (Get-Member -InputObject $ext -Name 'uBlock0@raymondhill.net' -MemberType NoteProperty) {
        $ext.'uBlock0@raymondhill.net' = $ublock
    } else {
        $ext | Add-Member -NotePropertyName 'uBlock0@raymondhill.net' -NotePropertyValue $ublock
    }
} else {
    $policies | Add-Member -NotePropertyName 'ExtensionSettings' -NotePropertyValue ([pscustomobject]@{
        'uBlock0@raymondhill.net' = $ublock
    })
}

# ── Write the policy file ────────────────────────────────────────────────────

$json = $data | ConvertTo-Json -Depth 10
# Normalize to LF line endings
$json = $json -replace "`r`n", "`n"

[System.IO.File]::WriteAllText($policyFile, "$json`n", [System.Text.UTF8Encoding]::new($false))
Write-Host "Policy written to $policyFile"

# ── Erase all bookmarks from existing profiles ──────────────────────────────
# Wipe every row from moz_bookmarks so Firefox starts clean with only the
# policy-defined folder. Each places.sqlite is backed up first.

$profileRoot = Join-Path $env:APPDATA 'Mozilla\Firefox\Profiles'
if (Test-Path $profileRoot) {
    $databases = Get-ChildItem -Path $profileRoot -Recurse -Filter 'places.sqlite' -ErrorAction SilentlyContinue

    if ($databases.Count -eq 0) {
        Write-Host 'No Firefox profile bookmark databases found to clean.'
    } else {
        foreach ($db in $databases) {
            $dbPath = $db.FullName
            $timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
            $dbBackup = "$dbPath.rock-os-backup.$timestamp"

            try {
                Copy-Item $dbPath $dbBackup
                # Use Python to wipe the SQLite table since PowerShell has
                # no built-in SQLite support.
                $pyScript = @"
import sqlite3, sys
conn = sqlite3.connect(sys.argv[1])
cur = conn.cursor()
cur.execute('DELETE FROM moz_bookmarks')
removed = cur.rowcount
conn.commit()
conn.close()
print(f'Erased {removed} bookmark(s) from {sys.argv[1]}')
"@
                python3 -c $pyScript "$dbPath" 2>$null
                if ($LASTEXITCODE -ne 0) {
                    # Try 'python' if 'python3' is not available (common on Windows)
                    python -c $pyScript "$dbPath"
                }
                Write-Host "Bookmark database backup saved to $dbBackup"
            } catch {
                Write-Host "Could not clean ${dbPath}: $_"
                Write-Host 'Close Firefox completely, then run this script again.'
            }
        }
    }
} else {
    Write-Host 'No Firefox profiles folder found. Skipping bookmark cleanup.'
}

# ── Done ─────────────────────────────────────────────────────────────────────

Write-Host ''
Write-Host 'Firefox policy installed:'
Write-Host $policyFile
Write-Host ''
Write-Host 'Restart Firefox, then open about:policies to verify it loaded.'

} catch {
    Write-Host ''
    Write-Host "ERROR: $_" -ForegroundColor Red
} finally {
    Write-Host ''
    Read-Host 'Press Enter to exit'
}

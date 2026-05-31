# HX99G-Firefox.ps1
# Installs Firefox via winget if missing, then configures it with a Rock-OS
# enterprise policy: privacy and utility extensions (uBlock Origin, Tabliss,
# Privacy Badger, CanvasBlocker, Multi-Account Containers, Skip Redirect,
# I Still Don't Care About Cookies, Startpage Search), always-visible
# bookmarks toolbar, and a set of toolbar bookmarks.
#
# Firefox reads enterprise policies from distribution\policies.json at startup.
# This script merges a small Rock-OS policy into that file instead of editing
# Firefox's profile database directly, which is safer and easier to review.
#
# WARNING: After confirmation, this script PERMANENTLY DELETES all Firefox data
# (every profile: bookmarks, history, saved passwords/logins, cookies, sessions,
# preferences, and extension data) for a clean, fresh start. There is NO backup.
# The policy itself survives the wipe because it lives in the install directory
# (distribution\policies.json), not in the profile, so the configured extensions
# and bookmarks are re-applied to the new profile on next launch.

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
Write-Host '  - DELETE all Firefox data (full profile reset: bookmarks, history,'
Write-Host '    saved passwords/logins, cookies, sessions, prefs, extension data)'
Write-Host '  - Add a bookmark folder to the toolbar'
Write-Host '  - Install uBlock Origin, Tabliss, Privacy Badger, CanvasBlocker,'
Write-Host '    Multi-Account Containers, Skip Redirect, I Still Don''t Care'
Write-Host '    About Cookies, and Startpage Search extensions'
Write-Host ''
Write-Host '========================================================================' -ForegroundColor Red
Write-Host '  WARNING: This PERMANENTLY DELETES ALL Firefox data for a fresh start.' -ForegroundColor Red
Write-Host '  That includes bookmarks, history, saved passwords/logins, cookies,'   -ForegroundColor Red
Write-Host '  open tabs/sessions, preferences, and extension data. NO BACKUP is'    -ForegroundColor Red
Write-Host '  made. If Firefox Sync is on, synced data may re-download afterward.'   -ForegroundColor Red
Write-Host '========================================================================' -ForegroundColor Red
Write-Host ''
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

# ── Close Firefox if running ─────────────────────────────────────────────────

$ffProcs = @(Get-Process -Name 'firefox' -ErrorAction SilentlyContinue)
if ($ffProcs.Count -gt 0) {
    Write-Host "Closing Firefox ($($ffProcs.Count) process(es))..."
    $ffProcs | Stop-Process -Force
    Start-Sleep -Seconds 2
    Write-Host 'Firefox closed.'
} else {
    Write-Host 'Firefox is not running.'
}
Write-Host ''

# ── Install Firefox via winget if not present ────────────────────────────────

$firefoxPaths = @(
    "$env:ProgramFiles\Mozilla Firefox\firefox.exe",
    "$env:ProgramFiles\Firefox Developer Edition\firefox.exe",
    "${env:ProgramFiles(x86)}\Mozilla Firefox\firefox.exe",
    "${env:ProgramFiles(x86)}\Firefox Developer Edition\firefox.exe",
    "$env:LOCALAPPDATA\Mozilla Firefox\firefox.exe",
    "$env:LOCALAPPDATA\Firefox Developer Edition\firefox.exe"
)

$firefoxDirs = @()
foreach ($path in $firefoxPaths) {
    if (Test-Path $path) {
        $dir = Split-Path $path -Parent
        if ($firefoxDirs -notcontains $dir) {
            $firefoxDirs += $dir
        }
    }
}

if ($firefoxDirs.Count -eq 0) {
    Write-Host 'Firefox not found. Installing via winget...'
    winget.exe install --id "Mozilla.Firefox" --exact --source winget `
        --accept-source-agreements --disable-interactivity --silent `
        --accept-package-agreements --force

    if ($LASTEXITCODE -ne 0) {
        Write-Host 'winget install failed. Install Firefox manually, then run this script again.'
        Read-Host 'Press Enter to exit'
        exit 1
    }

    # Refresh directories search after install
    foreach ($path in $firefoxPaths) {
        if (Test-Path $path) {
            $dir = Split-Path $path -Parent
            if ($firefoxDirs -notcontains $dir) {
                $firefoxDirs += $dir
            }
        }
    }

    if ($firefoxDirs.Count -eq 0) {
        Write-Host 'Firefox was installed but could not be found at the expected paths.'
        Write-Host 'You may need to restart your terminal or check the install location.'
        Read-Host 'Press Enter to exit'
        exit 1
    }
}

Write-Host 'Configuring policies for all discovered Firefox installations:'
foreach ($dir in $firefoxDirs) {
    Write-Host "  - $dir"
}
Write-Host ''

# ── Bookmarks ────────────────────────────────────────────────────────────────
# Each bookmark is a flat entry. The Folder field tells Firefox to group them
# inside a named folder on the toolbar. Firefox creates the folder automatically.

$folderName = "$([char]0x2B07)$([char]0xFE0F)Pirate"

$bookmarks = @(
    @{ Title = 'SkipVids';       URL = 'https://skipvids.com/';              Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'Ext';            URL = 'https://ext.to/';                    Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'TorrentGalaxy';  URL = 'https://torrentgalaxy.one/';         Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'PCGamesTorrent'; URL = 'https://pcgamestorrents.com/';       Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'Ziperto';        URL = 'https://www.ziperto.com/';           Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'DLPSGame';       URL = 'https://dlpsgame.com/category/ps4/'; Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'GetComics';      URL = 'https://getcomics.org/';             Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'PirateBay';      URL = 'https://thepiratebay10.xyz/';        Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'VibeMax';        URL = 'https://vibemax.to/';                Placement = 'toolbar'; Folder = $folderName }
)

# ── Extensions ────────────────────────────────────────────────────────────────

$extensions = [pscustomobject]@{
    'uBlock0@raymondhill.net' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi'
    }
    'extension@tabliss.io' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/tabliss/latest.xpi'
    }
    'jid1-MnnxcxisBPnSXQ@jetpack' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/privacy-badger17/latest.xpi'
    }
    'CanvasBlocker@kkapsner.de' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/canvasblocker/latest.xpi'
    }
    '@testpilot-containers' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/multi-account-containers/latest.xpi'
    }
    'skipredirect@sblask' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/skip-redirect/latest.xpi'
    }
    'idcac-pub@guus.ninja' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/istilldontcareaboutcookies/latest.xpi'
    }
    'StartpageSearchExtension@roteKlaue' = [pscustomobject]@{
        installation_mode = 'force_installed'
        install_url       = 'https://addons.mozilla.org/firefox/downloads/latest/startpage-search/latest.xpi'
    }
}

# ── Write policy files ───────────────────────────────────────────────────────

foreach ($firefoxDir in $firefoxDirs) {
    $policyDir  = Join-Path $firefoxDir 'distribution'
    $policyFile = Join-Path $policyDir 'policies.json'

    if (-not (Test-Path $policyDir)) {
        New-Item -ItemType Directory -Path $policyDir -Force | Out-Null
        Write-Host "Created policy directory: $policyDir"
    }

    # Load existing policy or start fresh
    if (Test-Path $policyFile) {
        $timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
        $backup = "$policyFile.rock-os-backup.$timestamp"
        Copy-Item $policyFile $backup
        Write-Host "Backed up existing policy to $backup"

        try {
            $data = Get-Content $policyFile -Raw -Encoding UTF8 | ConvertFrom-Json
        } catch {
            Write-Host "Existing policies.json at $policyFile was invalid, starting fresh."
            $data = [pscustomobject]@{}
        }
    } else {
        $data = [pscustomobject]@{}
    }

    # Build the policies object
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

    # Merge Bookmarks
    if (Get-Member -InputObject $policies -Name 'Bookmarks' -MemberType NoteProperty) {
        $policies.Bookmarks = $bookmarks
    } else {
        $policies | Add-Member -NotePropertyName 'Bookmarks' -NotePropertyValue $bookmarks
    }

    # Merge Extensions
    if (Get-Member -InputObject $policies -Name 'ExtensionSettings' -MemberType NoteProperty) {
        $policies.ExtensionSettings = $extensions
    } else {
        $policies | Add-Member -NotePropertyName 'ExtensionSettings' -NotePropertyValue $extensions
    }

    # Write
    $json = $data | ConvertTo-Json -Depth 10
    $json = $json -replace "`r`n", "`n"

    # Escape every non-ASCII character (e.g. the emoji in the bookmark folder
    # name) to a \uXXXX JSON escape so the output is pure ASCII. This makes the
    # file immune to codepage misreads: literal UTF-8 bytes can be decoded as
    # Windows-1252 and show up as mojibake (e.g. "â¬‡ï¸Pirate"), but \uXXXX
    # escapes decode to the correct characters no matter how the file is read.
    $sb = [System.Text.StringBuilder]::new()
    foreach ($ch in $json.ToCharArray()) {
        $code = [int][char]$ch
        if ($code -gt 127) {
            [void]$sb.AppendFormat('\u{0:x4}', $code)
        } else {
            [void]$sb.Append($ch)
        }
    }
    $json = $sb.ToString()

    [System.IO.File]::WriteAllText($policyFile, "$json`n", [System.Text.UTF8Encoding]::new($false))
    Write-Host "Policy written to $policyFile"
}

# ── Wipe ALL Firefox data (full profile reset) ──────────────────────────────
# Delete the entire Firefox data directory so Firefox starts completely fresh
# on next launch. This removes every profile (bookmarks, history, saved
# passwords/logins, cookies, sessions, prefs, extension data) and the cache.
#
# The enterprise policy (extensions + bookmarks) lives in
# Program Files\Mozilla Firefox\distribution\policies.json, NOT inside the
# profile, so it survives this wipe and is automatically applied to the brand
# new profile Firefox creates on next launch.
#
# This is a permanent delete with NO backup, per the script's configuration.

$firefoxDataRoots = @(
    (Join-Path $env:APPDATA      'Mozilla\Firefox'),
    (Join-Path $env:LOCALAPPDATA 'Mozilla\Firefox')
)

$wipedAny = $false
foreach ($dataRoot in $firefoxDataRoots) {
    if (Test-Path $dataRoot) {
        try {
            Remove-Item -LiteralPath $dataRoot -Recurse -Force -ErrorAction Stop
            Write-Host "Deleted $dataRoot"
            $wipedAny = $true
        } catch {
            Write-Host "Could not fully delete ${dataRoot}: $_"
            Write-Host 'Make sure Firefox is completely closed (check Task Manager for firefox.exe), then run this script again.'
        }
    }
}

if ($wipedAny) {
    Write-Host 'All Firefox data wiped. Firefox will create a fresh profile and apply the policy on next launch.'
} else {
    Write-Host 'No existing Firefox data found. Firefox will start fresh on next launch.'
}

# ── Done ─────────────────────────────────────────────────────────────────────

Write-Host ''
Write-Host 'Firefox policy installed for all discovered Firefox installations.'
Write-Host ''
Write-Host 'Restart Firefox, then open about:policies to verify it loaded.'

} catch {
    Write-Host ''
    Write-Host "ERROR: $_" -ForegroundColor Red
} finally {
    Write-Host ''
    Read-Host 'Press Enter to exit'
}

# HX99G-Firefox.ps1
# Installs Firefox via winget if missing, then configures it with a Rock-OS
# enterprise policy: privacy and utility extensions (uBlock Origin, Tabliss,
# Privacy Badger, CanvasBlocker, Multi-Account Containers, Skip Redirect,
# I Still Don't Care About Cookies), Startpage as the default search engine,
# always-visible bookmarks toolbar, and a set of toolbar bookmarks. It also
# applies these settings:
#   - Do not reopen previous tabs/windows on startup
#   - Confirm before closing a window with multiple tabs
#   - Enhanced Tracking Protection = Strict
#   - Global Privacy Control enabled ("don't sell or share my data")
#   - Max Protection secure DNS (DNS over HTTPS, no fallback)
#   - Turn off saving passwords, payment-method autofill, and address autofill
#   - Turn off all data collection: telemetry, studies, daily usage ping,
#     personalized extension recommendations, and remote feature rollouts
#   - Do not record browsing/download history, while keeping normal (non-private)
#     windows (the "custom settings for history" equivalent, NOT permanent private
#     browsing)
#   - Clear cookies and site data when Firefox is closed
#   - Turn off "Provide search suggestions" and "Remember search and form history"
#   - Turn off every Address Bar suggestion source (history, bookmarks, open tabs,
#     shortcuts, recent searches, search-engine suggestions, quick actions)
#   - Block AI enhancements (AIControls = blocked: chatbot, translations, etc.)
#
# Firefox reads enterprise policies from distribution\policies.json at startup.
# This script merges a small Rock-OS policy into that file instead of editing
# Firefox's profile database directly, which is safer and easier to review. A few
# preferences that have no matching policy (startup behavior, multi-tab warning,
# Global Privacy Control, Strict tracking protection, remote rollouts, daily
# usage ping) are applied via an AutoConfig file (rock-os.cfg) in the install
# directory, because the policies.json Preferences policy cannot set privacy.*
# prefs and cannot make ETP "Strict" stick.
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
Write-Host '    Multi-Account Containers, Skip Redirect, and I Still Don''t Care'
Write-Host '    About Cookies extensions'
Write-Host '  - Set Startpage as the default search engine'
Write-Host '  - Not reopen previous tabs/windows on startup'
Write-Host '  - Confirm before closing a window with multiple tabs'
Write-Host '  - Set Enhanced Tracking Protection to Strict'
Write-Host '  - Enable Global Privacy Control (tell sites not to sell/share data)'
Write-Host '  - Turn off saving passwords, payment info, and addresses'
Write-Host '  - Turn off all Firefox data collection (telemetry, studies, daily'
Write-Host '    usage ping, extension recommendations, remote rollouts)'
Write-Host '  - Enable Max Protection secure DNS (DNS over HTTPS)'
Write-Host '  - Do not record browsing/download history (normal, non-private windows)'
Write-Host '  - Clear cookies and site data when Firefox closes'
Write-Host '  - Turn off search suggestions'
Write-Host '  - Turn off remembering search and form history'
Write-Host '  - Turn off all Address Bar suggestions (history, bookmarks, open'
Write-Host '    tabs, shortcuts, recent searches, search engines, quick actions)'
Write-Host '  - Block AI enhancements (generative-AI features)'
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
# Each bookmark is a flat entry. A bookmark WITH a Folder field is grouped inside
# that named folder on the toolbar (Firefox creates the folder automatically);
# a bookmark WITHOUT a Folder field sits directly on the bookmarks toolbar.

$folderName = "$([char]0x2B07)$([char]0xFE0F)Pirate"

$bookmarks = @(
    # Directly on the bookmarks toolbar (no folder)
    @{ Title = 'SkipVids';       URL = 'https://skipvids.com/';              Placement = 'toolbar' },
    @{ Title = 'Jellyfin';       URL = 'http://192.168.1.53:8096';           Placement = 'toolbar' },

    # Inside the Pirate folder
    @{ Title = 'Ext';            URL = 'https://ext.to/';                    Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'TorrentGalaxy';  URL = 'https://torrentgalaxy.one/';         Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'PCGamesTorrent'; URL = 'https://pcgamestorrents.com/';       Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'Ziperto';        URL = 'https://www.ziperto.com/';           Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'DLPSGame';       URL = 'https://dlpsgame.com/category/ps4/'; Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'GetComics';      URL = 'https://getcomics.org/';             Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'PirateBay';      URL = 'https://thepiratebay10.xyz/';        Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'YTS';            URL = 'https://yts.hn/';                    Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'FMHY Torrenting';URL = 'https://fmhy.net/torrenting';        Placement = 'toolbar'; Folder = $folderName },
    @{ Title = 'Is It Cracked';  URL = 'https://isitcracked.com/';           Placement = 'toolbar'; Folder = $folderName }
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
}

# ── Search engine ──────────────────────────────────────────────────────────────
# Add Startpage as a real search engine and make it the default. Defining the
# engine directly (rather than relying on the Startpage add-on to register one)
# means the Default name always matches and there is no duplicate engine. The
# SearchEngines policy works on all Firefox release channels as of Firefox 139.

$searchEngines = [pscustomobject]@{
    Default = 'Startpage'
    Add     = @(
        [pscustomobject]@{
            Name               = 'Startpage'
            URLTemplate        = 'https://www.startpage.com/sp/search?query={searchTerms}'
            Method             = 'GET'
            SuggestURLTemplate = 'https://www.startpage.com/suggestions?q={searchTerms}'
        }
    )
}

# ── UserMessaging ───────────────────────────────────────────────────────────────
# Turn OFF "Allow personalized extension recommendations".
$userMessaging = [pscustomobject]@{
    ExtensionRecommendations = $false
}

# ── DNS over HTTPS (Max Protection) ─────────────────────────────────────────────
# Enabled + Fallback=false == "Max Protection": Firefox always uses secure DNS and
# shows a security-risk warning before falling back to system DNS. ProviderURL is
# Firefox's default (Cloudflare); change it if you prefer another resolver.
$dnsOverHttps = [pscustomobject]@{
    Enabled     = $true
    ProviderURL = 'https://mozilla.cloudflare-dns.com/dns-query'
    Fallback    = $false
    Locked      = $false
}

# ── AI controls (Block AI enhancements) ─────────────────────────────────────────
# Default = blocked turns off all generative-AI features (chatbot sidebar, page
# translations, PDF alt text, smart tab groups, link-preview key points, etc.).
# Requires Firefox 149+. Locked=false lets you re-enable individual features later.
$aiControls = [pscustomobject]@{
    Default = [pscustomobject]@{
        Value  = 'blocked'
        Locked = $false
    }
}

# ── Preferences (applied via AutoConfig, see "Write AutoConfig" below) ──────────
# These are written to a .cfg in the Firefox install directory rather than the
# policies.json "Preferences" policy, because that policy's allow-list does not
# include privacy.* prefs (needed for Global Privacy Control). AutoConfig can set
# any pref and, like policies.json, lives in the install dir so it survives the
# full profile wipe below.
#   browser.startup.page=1          -> do NOT reopen previous tabs/windows on start
#   browser.tabs.warnOnClose=true   -> confirm before closing a window with many tabs
#   browser.contentblocking.category=strict -> Enhanced Tracking Protection = Strict.
#       LOCKED on purpose: as a plain default pref Firefox recomputes the category
#       back to Standard, so locking is what makes Strict actually stick.
#   privacy.globalprivacycontrol.enabled=true -> "Tell websites not to sell/share my data"
#   app.normandy.enabled=false      -> OFF "improve features... between updates" (remote rollouts)
#   datareporting.usage.uploadEnabled=false -> OFF "Send daily usage ping to Mozilla"
#   places.history.enabled=false   -> do NOT record browsing/download history,
#       while still using normal (non-private) windows. This is the "custom
#       settings for history" equivalent. (The dropdown's "Never Remember History"
#       option is permanent PRIVATE browsing and forces every window private, which
#       is not what we want here.)
#   privacy.sanitize.sanitizeOnShutdown + clearOnShutdown(.cookies / _v2.cookiesAndStorage)
#       -> "Clear cookies and site data when Firefox is closed".
#   browser.search.suggest.enabled=false + browser.urlbar.suggest.searches=false
#       -> turn OFF "Provide search suggestions".
#   browser.formfill.enable=false   -> OFF "Remember search and form history".
#   browser.urlbar.suggest.* = false -> turn OFF every Address Bar suggestion source:
#       history (Browsing history), bookmark (Bookmarks), openpage (Open tabs),
#       topsites (Shortcuts), recentsearches (Recent searches),
#       engines (Suggest search engines to use), quickactions (Quick actions).

$prefLines = @(
    '// Rock-OS Firefox preferences (AutoConfig). First line is intentionally a comment.'
    'defaultPref("browser.startup.page", 1);'
    'defaultPref("browser.tabs.warnOnClose", true);'
    'lockPref("browser.contentblocking.category", "strict");'
    'defaultPref("privacy.globalprivacycontrol.enabled", true);'
    'defaultPref("app.normandy.enabled", false);'
    'defaultPref("datareporting.usage.uploadEnabled", false);'
    'defaultPref("places.history.enabled", false);'
    'defaultPref("privacy.sanitize.sanitizeOnShutdown", true);'
    'defaultPref("privacy.clearOnShutdown.cookies", true);'
    'defaultPref("privacy.clearOnShutdown_v2.cookiesAndStorage", true);'
    'defaultPref("browser.search.suggest.enabled", false);'
    'defaultPref("browser.formfill.enable", false);'
    'defaultPref("browser.urlbar.suggest.searches", false);'
    'defaultPref("browser.urlbar.suggest.history", false);'
    'defaultPref("browser.urlbar.suggest.bookmark", false);'
    'defaultPref("browser.urlbar.suggest.openpage", false);'
    'defaultPref("browser.urlbar.suggest.topsites", false);'
    'defaultPref("browser.urlbar.suggest.recentsearches", false);'
    'defaultPref("browser.urlbar.suggest.engines", false);'
    'defaultPref("browser.urlbar.suggest.quickactions", false);'
)
$prefCfg = ($prefLines -join "`n") + "`n"

$autoConfigLines = @(
    '// Rock-OS AutoConfig loader'
    'pref("general.config.filename", "rock-os.cfg");'
    'pref("general.config.obscure_value", 0);'
)
$autoConfigJs = ($autoConfigLines -join "`n") + "`n"

# ── Write policy files ───────────────────────────────────────────────────────

foreach ($firefoxDir in $firefoxDirs) {
    $policyDir  = Join-Path $firefoxDir 'distribution'
    $policyFile = Join-Path $policyDir 'policies.json'

    if (-not (Test-Path $policyDir)) {
        New-Item -ItemType Directory -Path $policyDir -Force | Out-Null
        Write-Host "Created policy directory: $policyDir"
    }

    # Load existing policy or start fresh (no backup is kept; the Rock-OS policy
    # is regenerated from this script, so the previous file is simply overwritten).
    if (Test-Path $policyFile) {
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

    # Simple on/off policies:
    #   DisplayBookmarksToolbar  - always show the bookmarks toolbar
    #   NoDefaultBookmarks       - don't create Firefox's default bookmarks
    #   DisableProfileImport     - suppress the import-from-another-browser prompt
    #   OfferToSaveLogins        - turn OFF "Ask to save passwords"
    #   AutofillAddressEnabled   - turn OFF "Save and autofill addresses"
    #   AutofillCreditCardEnabled- turn OFF "Save and autofill payment info"
    #   DisableTelemetry         - turn OFF "Send technical and interaction data"
    #   DisableFirefoxStudies    - turn OFF "Allow Firefox to run feature studies"
    $fieldDefaults = @{
        DisplayBookmarksToolbar   = $true
        NoDefaultBookmarks        = $true
        DisableProfileImport      = $true
        OfferToSaveLogins         = $false
        AutofillAddressEnabled    = $false
        AutofillCreditCardEnabled = $false
        DisableTelemetry          = $true
        DisableFirefoxStudies     = $true
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

    # Merge SearchEngines (Startpage as default)
    if (Get-Member -InputObject $policies -Name 'SearchEngines' -MemberType NoteProperty) {
        $policies.SearchEngines = $searchEngines
    } else {
        $policies | Add-Member -NotePropertyName 'SearchEngines' -NotePropertyValue $searchEngines
    }

    # Merge UserMessaging (no personalized extension recommendations)
    if (Get-Member -InputObject $policies -Name 'UserMessaging' -MemberType NoteProperty) {
        $policies.UserMessaging = $userMessaging
    } else {
        $policies | Add-Member -NotePropertyName 'UserMessaging' -NotePropertyValue $userMessaging
    }

    # Merge DNSOverHTTPS (Max Protection)
    if (Get-Member -InputObject $policies -Name 'DNSOverHTTPS' -MemberType NoteProperty) {
        $policies.DNSOverHTTPS = $dnsOverHttps
    } else {
        $policies | Add-Member -NotePropertyName 'DNSOverHTTPS' -NotePropertyValue $dnsOverHttps
    }

    # Merge AIControls (block AI enhancements)
    if (Get-Member -InputObject $policies -Name 'AIControls' -MemberType NoteProperty) {
        $policies.AIControls = $aiControls
    } else {
        $policies | Add-Member -NotePropertyName 'AIControls' -NotePropertyValue $aiControls
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

    # ── Write AutoConfig preference files ───────────────────────────────────
    # rock-os.cfg holds the prefs; defaults\pref\autoconfig.js tells Firefox to
    # load it. Both live in the install dir, so they survive the profile wipe.
    $cfgFile     = Join-Path $firefoxDir 'rock-os.cfg'
    $autoCfgDir  = Join-Path $firefoxDir 'defaults\pref'
    $autoCfgFile = Join-Path $autoCfgDir 'autoconfig.js'

    if (-not (Test-Path $autoCfgDir)) {
        New-Item -ItemType Directory -Path $autoCfgDir -Force | Out-Null
    }

    $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText($cfgFile, $prefCfg, $utf8NoBom)
    [System.IO.File]::WriteAllText($autoCfgFile, $autoConfigJs, $utf8NoBom)
    Write-Host "Preferences written to $cfgFile"
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

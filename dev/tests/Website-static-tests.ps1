# Website-static-tests.ps1
# Lightweight static checks for Rock-OS website files. This does not start the
# Go server, run browser automation, or publish anything.

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path     # dev/tests
$RepoRoot  = Split-Path -Parent (Split-Path -Parent $ScriptDir)  # repo root
$WebsiteDir = Join-Path $RepoRoot 'Website'

$ErrorActionPreference = 'Stop'
$Failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $Failures.Add($Message)
    Write-Host "[FAIL] $Message" -ForegroundColor Red
}

function Write-Check {
    param([string]$Message)
    Write-Host "[CHECK] $Message" -ForegroundColor Gray
}

function Write-OK {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Test-RepoRelativePathExists {
    param(
        [string]$BaseDir,
        [string]$Reference,
        [string]$Source
    )

    if ([string]::IsNullOrWhiteSpace($Reference)) {
        return
    }
    if ($Reference -match '^(https?:|mailto:|tel:|data:|#|//)') {
        return
    }

    $cleanReference = ($Reference -split '[?#]', 2)[0]
    if ([string]::IsNullOrWhiteSpace($cleanReference)) {
        return
    }

    $candidate = if ($cleanReference.StartsWith('/')) {
        Join-Path $WebsiteDir $cleanReference.TrimStart('/')
    } else {
        Join-Path $BaseDir $cleanReference
    }

    if (-not (Test-Path -LiteralPath $candidate)) {
        Add-Failure "$Source references missing local file: $Reference"
    }
}

Write-Host "Rock-OS website static tests - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Write-Host

if (-not (Test-Path -LiteralPath $WebsiteDir)) {
    Add-Failure "Website folder not found: $WebsiteDir"
}

Write-Check "JavaScript syntax"
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Add-Failure 'node command not found. Install Node.js or skip website JS syntax checks.'
} else {
    $jsFiles = Get-ChildItem -LiteralPath (Join-Path $WebsiteDir 'js') -Recurse -File -Filter '*.js' |
        Where-Object { $_.FullName -notmatch '\\vendor\\' }

    foreach ($file in $jsFiles) {
        $output = & node --check $file.FullName 2>&1
        if ($LASTEXITCODE -ne 0) {
            Add-Failure "JavaScript syntax failed: $($file.FullName)"
            $output | ForEach-Object { Write-Host "  $_" -ForegroundColor Red }
        }
    }

    if ($Failures.Count -eq 0) {
        Write-OK "Checked $($jsFiles.Count) JavaScript files."
    }
}

Write-Check "JSON validity"
$jsonFiles = @()
$jsonFiles += Get-ChildItem -LiteralPath (Join-Path $WebsiteDir 'ENCRYPTED\Sessions') -Recurse -File -Filter 'dashboard.json' -ErrorAction SilentlyContinue
$jsonFiles += Get-Item -LiteralPath (Join-Path $WebsiteDir 'Sessions-State\sessions.json') -ErrorAction SilentlyContinue
$jsonFiles += Get-Item -LiteralPath (Join-Path $WebsiteDir 'site.webmanifest') -ErrorAction SilentlyContinue

foreach ($file in $jsonFiles) {
    try {
        Get-Content -Raw -LiteralPath $file.FullName | ConvertFrom-Json | Out-Null
    } catch {
        Add-Failure "Invalid JSON: $($file.FullName) - $($_.Exception.Message)"
    }
}
Write-OK "Parsed $($jsonFiles.Count) JSON files."

Write-Check "Public HTML local references"
$htmlFiles = Get-ChildItem -LiteralPath $WebsiteDir -File -Filter '*.html'
foreach ($file in $htmlFiles) {
    $html = Get-Content -Raw -LiteralPath $file.FullName
    $baseDir = Split-Path -Parent $file.FullName

    foreach ($match in [regex]::Matches($html, '<script\b[^>]*\bsrc="([^"]+)"')) {
        Test-RepoRelativePathExists -BaseDir $baseDir -Reference $match.Groups[1].Value -Source $file.FullName
    }
    foreach ($match in [regex]::Matches($html, '<link\b[^>]*\bhref="([^"]+)"')) {
        Test-RepoRelativePathExists -BaseDir $baseDir -Reference $match.Groups[1].Value -Source $file.FullName
    }
    foreach ($match in [regex]::Matches($html, '<img\b[^>]*\bsrc="([^"]+)"')) {
        Test-RepoRelativePathExists -BaseDir $baseDir -Reference $match.Groups[1].Value -Source $file.FullName
    }
}
Write-OK "Checked $($htmlFiles.Count) public HTML files."

Write-Check "CSS local URL references"
$cssFiles = Get-ChildItem -LiteralPath (Join-Path $WebsiteDir 'css') -Recurse -File -Filter '*.css'
foreach ($file in $cssFiles) {
    $css = Get-Content -Raw -LiteralPath $file.FullName
    $baseDir = Split-Path -Parent $file.FullName

    foreach ($match in [regex]::Matches($css, 'url\(\s*[''"]?([^''")]+)[''"]?\s*\)')) {
        Test-RepoRelativePathExists -BaseDir $baseDir -Reference $match.Groups[1].Value -Source $file.FullName
    }
}
Write-OK "Checked $($cssFiles.Count) CSS files."

Write-Check "Retired path scan"
$retiredPatterns = @(
    'Website/ENCRYPTED/menu',
    'ENCRYPTED/menu',
    'guides.html',
    'api/guides',
    'guides-index',
    'launch-point-cards-locked',
    'locked-landing'
)
$scanRoots = @(
    (Join-Path $WebsiteDir 'js'),
    (Join-Path $WebsiteDir 'css'),
    $htmlFiles.FullName,
    (Join-Path $RepoRoot 'README.md'),
    (Join-Path $RepoRoot 'AGENTS.md')
) | Where-Object { $_ -and (Test-Path -LiteralPath $_) }

foreach ($root in $scanRoots) {
    $files = if ((Get-Item -LiteralPath $root).PSIsContainer) {
        Get-ChildItem -LiteralPath $root -Recurse -File | Where-Object { $_.FullName -notmatch '\\vendor\\' }
    } else {
        @(Get-Item -LiteralPath $root)
    }

    foreach ($file in $files) {
        $text = Get-Content -Raw -LiteralPath $file.FullName
        foreach ($pattern in $retiredPatterns) {
            if ($text.Contains($pattern)) {
                Add-Failure "$($file.FullName) contains retired path/reference: $pattern"
            }
        }
    }
}
Write-OK "Retired path scan complete."

Write-Check "Nested session profile path parsing"
$pathParserChecks = @(
    @{
        File = Join-Path $WebsiteDir 'js\profiles.js'
        Pattern = "parts[3] !== 'Profiles'"
    },
    @{
        File = Join-Path $WebsiteDir 'js\wiki\links.js'
        Pattern = "parts[3] === 'Profiles'"
    }
)
foreach ($check in $pathParserChecks) {
    if (Test-Path -LiteralPath $check.File) {
        $text = Get-Content -Raw -LiteralPath $check.File
        if ($text.Contains($check.Pattern)) {
            Add-Failure "$($check.File) still assumes Profiles is at a fixed session path segment."
        }
    }
}
Write-OK "Nested session profile path parsing checks complete."

Write-Check "Git whitespace check"
if (Get-Command git -ErrorAction SilentlyContinue) {
    Push-Location $RepoRoot
    try {
        $output = & git diff --check 2>&1
        if ($LASTEXITCODE -ne 0) {
            Add-Failure 'git diff --check failed.'
            $output | ForEach-Object { Write-Host "  $_" -ForegroundColor Red }
        } else {
            Write-OK 'git diff --check passed.'
        }
    } finally {
        Pop-Location
    }
} else {
    Add-Failure 'git command not found. Could not run git diff --check.'
}

Write-Host
if ($Failures.Count -gt 0) {
    Write-Host "RESULT: FAIL ($($Failures.Count) issue(s))" -ForegroundColor Red
    exit 1
}

Write-Host 'RESULT: PASS' -ForegroundColor Green

# Rock-OS Release Creation Script
# Automates checks, cross-compilation, and checksum generation for new releases.

param(
    [string]$Version,
    [switch]$Publish,
    [switch]$SkipPublish
)

function Wait-ForReleaseExit {
    if ([Environment]::UserInteractive) {
        Write-Host
        Read-Host "Press Enter to exit"
    }
}

function Exit-Release {
    param(
        [int]$Code = 0
    )

    Wait-ForReleaseExit
    Exit $Code
}

if ($Publish -and $SkipPublish) {
    Write-Host "[ERROR] Use either -Publish or -SkipPublish, not both." -ForegroundColor Red
    Exit-Release 1
}

$shouldPublish = -not $SkipPublish

# Ensure we run from the repo root
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptPath "..")
Set-Location $repoRoot
$goCache = Join-Path $repoRoot ".gotest-cache"
New-Item -ItemType Directory -Path $goCache -Force | Out-Null
$env:GOCACHE = $goCache

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "       Rock-OS Release Builder" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host

# 1. Check if git is available
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] git command not found. Please install git and add it to your PATH." -ForegroundColor Red
    Exit-Release 1
}

# 2. Check if go is available
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] go command not found. Go is required to compile release binaries." -ForegroundColor Red
    Exit-Release 1
}

# 3. Run server checks before building release binaries
Write-Host "Running Go server checks..." -ForegroundColor Gray
$goTestScript = Join-Path $repoRoot "dev\tests\GO-tests.ps1"
& powershell -NoProfile -ExecutionPolicy Bypass -File $goTestScript
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Go server checks failed. Release build stopped." -ForegroundColor Red
    Exit-Release 1
}
Write-Host "[OK] Go server checks passed." -ForegroundColor Green

# 4. Run website static checks before building release binaries
Write-Host "Running website static checks..." -ForegroundColor Gray
$websiteStaticTest = Join-Path $repoRoot "dev\tests\Website-static-tests.ps1"
& powershell -NoProfile -ExecutionPolicy Bypass -File $websiteStaticTest
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Website static checks failed. Release build stopped." -ForegroundColor Red
    Exit-Release 1
}
Write-Host "[OK] Website static checks passed." -ForegroundColor Green

# 5. Ask for next version interactively unless supplied by the caller
$currentVersion = "none"
$suggestionPrompt = "e.g., 1.0 or 1.1"

Write-Host "Fetching latest release version from GitHub..." -ForegroundColor Gray
try {
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/rocketpowerinc/rock-os/releases/latest" -TimeoutSec 5 -ErrorAction Stop
    if ($response -and $response.tag_name) {
        $tag = $response.tag_name.Trim()
        $verStr = $tag
        if ($verStr.StartsWith("v")) {
            $verStr = $verStr.Substring(1)
        }
        if ($verStr -notmatch "\.") {
            $verStr = "$verStr.0"
        }
        $verObj = [version]$verStr
        $currentVersion = $tag

        if ($verObj.Build -ge 0) {
            $suggestPatch = "$($verObj.Major).$($verObj.Minor).$($verObj.Build + 1)"
            $suggestMinor = "$($verObj.Major).$($verObj.Minor + 1)"
            $suggestionPrompt = "Current GitHub: $currentVersion, suggest $suggestPatch or $suggestMinor"
        } else {
            $suggestPoint = "$($verObj.Major).$($verObj.Minor + 1)"
            $suggestMajor = "$($verObj.Major + 1).0"
            $suggestionPrompt = "Current GitHub: $currentVersion, suggest $suggestPoint or $suggestMajor"
        }
    }
} catch {
    Write-Host "[ERROR] Could not fetch the latest release version from GitHub: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Please check your internet connection and verify that the repository is public." -ForegroundColor Yellow
    Exit-Release 1
}

$rawVersion = $Version
if (-not $rawVersion) {
    $rawVersion = Read-Host "Enter the next version number ($suggestionPrompt)"
}
if (-not $rawVersion) {
    Write-Host "[ERROR] Version number cannot be empty." -ForegroundColor Red
    Exit-Release 1
}

# Normalize version (strip leading 'v' or 'v.' if present, then build clean vX.Y)
$cleanVersion = $rawVersion.ToLower().Trim()
if ($cleanVersion.StartsWith("v")) {
    $cleanVersion = $cleanVersion.Substring(1)
}
if ($cleanVersion.StartsWith(".")) {
    $cleanVersion = $cleanVersion.Substring(1)
}
if ($cleanVersion -notmatch "^\d+\.\d+(\.\d+)?$") {
    Write-Host "[ERROR] Version must look like 8.0 or 8.0.1." -ForegroundColor Red
    Exit-Release 1
}

$versionName = "v$cleanVersion"
$releaseDir = Join-Path ".release" $versionName

# 5. Stage and commit pending release source changes
Write-Host
Write-Host "Preparing release commit..." -ForegroundColor Gray
git diff --cached --quiet
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] The Git index already contains staged changes. Unstage them before creating a release." -ForegroundColor Red
    Exit-Release 1
}

git add --all
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Could not stage release changes." -ForegroundColor Red
    Exit-Release 1
}

$stagedFiles = @(git diff --cached --name-only)
$forbiddenPatterns = @(
    '(^|/).*\.key$',
    '^Website/\.rock-os-version$',
    '^Website/\.rock-os-wiki-version$',
    '^Website/Sessions/active-session\.json$',
    '^Website/(markdown|wiki|bootstraps|cheatsheets|dotfiles|bookmarks|profiles|dashboards)-index\.json$',
    '^Website/rock-os-',
    '^Website/.*\.download$',
    '(^|/)\.gocache/',
    '(^|/)\.gotest-cache/',
    '^\.release/'
)

$forbiddenFiles = @(
    $stagedFiles | Where-Object {
        $path = $_
        $forbiddenPatterns | Where-Object { $path -match $_ }
    }
)

if ($forbiddenFiles.Count -gt 0) {
    Write-Host "[ERROR] Refusing to commit generated artifacts or secrets:" -ForegroundColor Red
    $forbiddenFiles | ForEach-Object { Write-Host " - $_" -ForegroundColor Red }
    git reset --mixed HEAD
    Exit-Release 1
}

git diff --cached --check
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Staged changes failed the whitespace check." -ForegroundColor Red
    git reset --mixed HEAD
    Exit-Release 1
}

if ($stagedFiles.Count -gt 0) {
    git commit -m "release: prepare $versionName"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Could not create the release commit." -ForegroundColor Red
        git reset --mixed HEAD
        Exit-Release 1
    }
    Write-Host "[OK] Created release commit for $versionName." -ForegroundColor Green
} else {
    Write-Host "[OK] No pending source changes to commit." -ForegroundColor Green
}

Write-Host
Write-Host "Preparing to build $versionName release into: $releaseDir" -ForegroundColor Cyan
Write-Host

# Create release directory
if (-not (Test-Path $releaseDir)) {
    New-Item -ItemType Directory -Path $releaseDir | Out-Null
}

$absoluteReleaseDir = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $releaseDir))

$targets = @(
    @{ os = "windows"; arch = "amd64"; suffix = ".exe"; name = "rock-os-windows-amd64.exe" },
    @{ os = "windows"; arch = "arm64"; suffix = ".exe"; name = "rock-os-windows-arm64.exe" },
    @{ os = "linux";   arch = "amd64"; suffix = "";     name = "rock-os-linux-amd64" },
    @{ os = "linux";   arch = "arm64"; suffix = "";     name = "rock-os-linux-arm64" },
    @{ os = "darwin";  arch = "amd64"; suffix = "";     name = "rock-os-macos-amd64" },
    @{ os = "darwin";  arch = "arm64"; suffix = "";     name = "rock-os-macos-arm64" }
)

$checksums = @()

# Move into module directory to compile with go.mod present
Push-Location "cmd/rock-os"

try {
    foreach ($target in $targets) {
        $os = $target.os
        $arch = $target.arch
        $binaryName = $target.name
        $outputPath = Join-Path $absoluteReleaseDir $binaryName

        Write-Host "Building $os/$arch -> $binaryName..." -ForegroundColor Gray

        # Set environment variables for cross-compilation
        $env:GOOS = $os
        $env:GOARCH = $arch

        # Compile binary with optimizations
        go build -ldflags="-s -w" -o $outputPath .

        if ($LASTEXITCODE -ne 0) {
            Write-Host "[ERROR] Failed to compile binary for $os/$arch" -ForegroundColor Red
            Exit-Release 1
        }

        # Generate checksum
        $hash = (Get-FileHash -Path $outputPath -Algorithm SHA256).Hash.ToLower()
        $checksumLine = "$hash  $binaryName"
        $checksums += $checksumLine
    }
} finally {
    Remove-Item env:GOOS -ErrorAction SilentlyContinue
    Remove-Item env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

# Write checksums file
$checksumFile = Join-Path $releaseDir "rock-os-$versionName-checksums.txt"
$checksums | Out-File -FilePath $checksumFile -Encoding ascii

Write-Host
Write-Host "==========================================" -ForegroundColor Green
Write-Host "   Release $versionName created successfully!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host
Write-Host "Release files list:" -ForegroundColor Gray
Get-ChildItem $releaseDir | ForEach-Object {
    Write-Host " - $_" -ForegroundColor Cyan
}

Write-Host
if ($shouldPublish) {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Host "[ERROR] gh command not found. Please install GitHub CLI to publish releases." -ForegroundColor Red
        Exit-Release 1
    }

    $currentBranch = git branch --show-current
    Write-Host "Pushing current branch ($currentBranch) to origin..." -ForegroundColor Gray
    git push origin $currentBranch
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to push commits to GitHub." -ForegroundColor Red
        Exit-Release 1
    }

    Write-Host "Creating GitHub release $versionName and uploading assets..." -ForegroundColor Gray
    $filesToUpload = (Get-ChildItem -Path (Join-Path $absoluteReleaseDir "*") -File).FullName
    gh release create $versionName $filesToUpload --title "$versionName" --notes "Release $versionName"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to publish release to GitHub." -ForegroundColor Red
        Exit-Release 1
    }

    Write-Host "[OK] Release published successfully to GitHub!" -ForegroundColor Green
} else {
    Write-Host "Skipped publishing to GitHub. Binaries are prepared locally in $releaseDir." -ForegroundColor Yellow
    Write-Host "Re-run with -SkipPublish only when you explicitly want a local-only build. The script will prompt for the version." -ForegroundColor Yellow
}

Write-Host
Write-Host "Done." -ForegroundColor Green
Exit-Release 0

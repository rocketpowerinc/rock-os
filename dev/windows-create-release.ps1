# Rock-OS Release Creation Script
# Automates checks, cross-compilation, and checksum generation for new releases.

# Ensure we run from the repo root
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptPath "..")
Set-Location $repoRoot

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "       Rock-OS Release Builder" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host

# 1. Check if git is available
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] git command not found. Please install git and add it to your PATH." -ForegroundColor Red
    Exit 1
}

# 2. Check if repo is clean
Write-Host "Checking repository status..." -ForegroundColor Gray
$gitStatus = git status --porcelain
if ($gitStatus) {
    Write-Host "[ERROR] Your repository has uncommitted changes. Please commit or stash them first." -ForegroundColor Red
    Write-Host $gitStatus
    Exit 1
}
Write-Host "[OK] Repository is clean." -ForegroundColor Green

# 3. Check if go is available
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] go command not found. Go is required to compile release binaries." -ForegroundColor Red
    Exit 1
}

# 4. Ask for next version interactively
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
    Write-Host "Warning: Could not fetch latest release from GitHub ($($_.Exception.Message))." -ForegroundColor Yellow
    # Fallback to local .release directory scanning
    if (Test-Path ".release") {
        $dirs = Get-ChildItem ".release" -Directory | Where-Object { $_.Name -match "^v\d+(\.\d+)*$" }
        if ($dirs) {
            $versions = @()
            foreach ($dir in $dirs) {
                $verStr = $dir.Name.Substring(1)
                if ($verStr -notmatch "\.") {
                    $verStr = "$verStr.0"
                }
                try {
                    $verObj = [version]$verStr
                    $versions += [PSCustomObject]@{
                        OriginalName = $dir.Name
                        VersionObj   = $verObj
                    }
                } catch {}
            }
            if ($versions) {
                $latestEntry = $versions | Sort-Object VersionObj | Select-Object -Last 1
                $currentVersion = $latestEntry.OriginalName
                $latestVer = $latestEntry.VersionObj

                if ($latestVer.Build -ge 0) {
                    $suggestPatch = "$($latestVer.Major).$($latestVer.Minor).$($latestVer.Build + 1)"
                    $suggestMinor = "$($latestVer.Major).$($latestVer.Minor + 1)"
                    $suggestionPrompt = "Current local: $currentVersion, suggest $suggestPatch or $suggestMinor"
                } else {
                    $suggestPoint = "$($latestVer.Major).$($latestVer.Minor + 1)"
                    $suggestMajor = "$($latestVer.Major + 1).0"
                    $suggestionPrompt = "Current local: $currentVersion, suggest $suggestPoint or $suggestMajor"
                }
            }
        }
    }
}

$rawVersion = Read-Host "Enter the next version number ($suggestionPrompt)"
if (-not $rawVersion) {
    Write-Host "[ERROR] Version number cannot be empty." -ForegroundColor Red
    Exit 1
}

# Normalize version (strip leading 'v' or 'v.' if present, then build clean vX.Y)
$cleanVersion = $rawVersion.ToLower().Trim()
if ($cleanVersion.StartsWith("v")) {
    $cleanVersion = $cleanVersion.Substring(1)
}
if ($cleanVersion.StartsWith(".")) {
    $cleanVersion = $cleanVersion.Substring(1)
}

$versionName = "v$cleanVersion"
$releaseDir = Join-Path ".release" $versionName

Write-Host
Write-Host "Preparing to build $versionName release into: $releaseDir" -ForegroundColor Cyan
Write-Host

# Create release directory
if (-not (Test-Path $releaseDir)) {
    New-Item -ItemType Directory -Path $releaseDir | Out-Null
}

$targets = @(
    @{ os = "windows"; arch = "amd64"; suffix = ".exe"; name = "rock-os-wiki-windows-amd64.exe" },
    @{ os = "windows"; arch = "arm64"; suffix = ".exe"; name = "rock-os-wiki-windows-arm64.exe" },
    @{ os = "linux";   arch = "amd64"; suffix = "";     name = "rock-os-wiki-linux-amd64" },
    @{ os = "linux";   arch = "arm64"; suffix = "";     name = "rock-os-wiki-linux-arm64" },
    @{ os = "darwin";  arch = "amd64"; suffix = "";     name = "rock-os-wiki-macos-amd64" },
    @{ os = "darwin";  arch = "arm64"; suffix = "";     name = "rock-os-wiki-macos-arm64" }
)

$checksums = @()

foreach ($target in $targets) {
    $os = $target.os
    $arch = $target.arch
    $binaryName = $target.name
    $outputPath = Join-Path $releaseDir $binaryName

    Write-Host "Building $os/$arch -> $binaryName..." -ForegroundColor Gray

    # Set environment variables for cross-compilation
    $env:GOOS = $os
    $env:GOARCH = $arch

    # Compile binary with optimizations
    go build -ldflags="-s -w" -o $outputPath cmd/rock-os-wiki/main.go
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to compile binary for $os/$arch" -ForegroundColor Red
        Exit 1
    }

    # Generate checksum
    $hash = (Get-FileHash -Path $outputPath -Algorithm SHA256).Hash.ToLower()
    $checksumLine = "$hash  $binaryName"
    $checksums += $checksumLine
}

# Write checksums file
$checksumFile = Join-Path $releaseDir "rock-os-wiki-$versionName-checksums.txt"
$checksums | Out-File -FilePath $checksumFile -Encoding ascii

# Reset environment variables to default
Remove-Item env:GOOS -ErrorAction SilentlyContinue
Remove-Item env:GOARCH -ErrorAction SilentlyContinue

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
Write-Host "Done." -ForegroundColor Green

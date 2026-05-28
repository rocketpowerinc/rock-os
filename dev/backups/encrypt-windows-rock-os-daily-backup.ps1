<#
    encrypt-windows-rock-os-daily-backup.ps1

    Creates a FULL snapshot ZIP of the Rock-OS repo (including the .git folder,
    the git-crypt .key, and any uncommitted changes) and encrypts it with
    OpenSSL AES-256-CBC. An HMAC-SHA256 sidecar (.hmac) is written alongside the
    .enc so the decrypt script can detect corruption or tampering before
    decrypting.

    The encrypted backup is written to the user's Downloads folder.

    IMPORTANT:
    This backup contains the git-crypt .key and your decrypted Profiles.
    Handle the .enc (and ESPECIALLY any decrypted output) with extreme care.

    NOTE:
    This script does NOT pull, does NOT require a clean working tree, and does
    NOT modify the repo in any way. It snapshots whatever is on disk right now.
#>

$ErrorActionPreference = 'Stop'

# This script is meant to run from dev\backups\ inside your working clone.
# It resolves dev\backups\..\..  to the repo root.
$repoPath = (Resolve-Path (Join-Path $PSScriptRoot "..\..") -ErrorAction SilentlyContinue).Path
if (-not $repoPath -or -not (Test-Path (Join-Path $repoPath '.git'))) {
    Write-Host "ERROR: Run this from inside the Rock-OS development clone (dev\backups\)." -ForegroundColor Red
    Write-Host "       It cannot be run from an arbitrary location." -ForegroundColor Red
    exit 1
}

$backupDir = Join-Path $env:USERPROFILE "Downloads"
$timestamp = Get-Date -Format "MMM-d-yyyy_h-mmtt"
$encPath   = Join-Path $backupDir "rock-os-backup-$timestamp.zip.enc"
$macPath   = "$encPath.hmac"

# Build the ZIP in a NON-synced temp dir so a plaintext copy never lands in
# Downloads (which is often cloud-synced by OneDrive/Dropbox). Only the
# encrypted .enc is ever written to Downloads.
$tempZip = Join-Path $env:TEMP "rock-os-backup-$PID.zip"

function Fail($Message) {
    Write-Host "ERROR: $Message" -ForegroundColor Red
    exit 1
}
function Test-Command($Name) { return [bool](Get-Command $Name -ErrorAction SilentlyContinue) }

if (-not (Test-Command openssl)) { Fail "OpenSSL is not installed or not available in PATH." }

# Folders inside the repo to skip: regenerable build caches. Edit this list if
# you want a leaner or fuller snapshot. Paths are relative to the repo root.
$excludeDirs = @(
    'Website\.gocache',
    'Website\.gotmp'
)

Add-Type -AssemblyName System.IO.Compression.FileSystem

Write-Host "Snapshotting repo (including .git, .key, and uncommitted changes)..." -ForegroundColor Cyan

# Zip DIRECTLY from the repo - no staging copy. This avoids the MAX_PATH
# doubling you'd hit by copying into %TEMP% first. Long source paths are read
# with the \\?\ extended-length prefix.
$repoFull = (Resolve-Path -LiteralPath $repoPath).Path.TrimEnd('\')
$allFiles = [System.IO.Directory]::EnumerateFiles($repoFull, '*', [System.IO.SearchOption]::AllDirectories)

if (Test-Path -LiteralPath $tempZip) { Remove-Item -LiteralPath $tempZip -Force }

$skipped = 0
$zip = [System.IO.Compression.ZipFile]::Open($tempZip, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($file in $allFiles) {
        $rel = $file.Substring($repoFull.Length + 1)

        $skip = $false
        foreach ($ex in $excludeDirs) {
            if ($rel -eq $ex -or $rel.StartsWith($ex + '\')) { $skip = $true; break }
        }
        if ($skip) { continue }

        $entryName = $rel -replace '\\', '/'
        $srcPath = $file
        if ($srcPath.Length -ge 248 -and -not $srcPath.StartsWith('\\?\')) {
            $srcPath = '\\?\' + $srcPath
        }

        try {
            [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
                $zip, $srcPath, $entryName,
                [System.IO.Compression.CompressionLevel]::Optimal) | Out-Null
        } catch {
            $skipped++
            Write-Host "  WARNING: skipped $rel ($($_.Exception.Message))" -ForegroundColor Yellow
        }
    }
}
finally {
    $zip.Dispose()
}

if ($skipped -gt 0) {
    Write-Host "$skipped file(s) could not be added (likely locked/in-use)." -ForegroundColor Yellow
}
Write-Host "Snapshot ZIP staged at $tempZip" -ForegroundColor Green

$plain = $null
try {
    # --- Secure password input with confirmation ---
    $match = $false
    while (-not $match) {
        $password = Read-Host "Enter encryption password" -AsSecureString
        $confirm  = Read-Host "Confirm encryption password" -AsSecureString

        $pBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
        $cBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($confirm)
        $plain        = [Runtime.InteropServices.Marshal]::PtrToStringAuto($pBstr)
        $plainConfirm = [Runtime.InteropServices.Marshal]::PtrToStringAuto($cBstr)
        $match = ($plain -eq $plainConfirm)
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pBstr)
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($cBstr)
        $plainConfirm = $null

        if (-not $match) { Write-Host "Passwords do not match. Please try again." -ForegroundColor Yellow }
    }

    # --- Encrypt. Password is piped via stdin and never written to disk. ---
    Write-Host "Encrypting with AES-256-CBC (pbkdf2, 600k iterations, sha256)..." -ForegroundColor Cyan
    $plain | & openssl enc -aes-256-cbc -salt -pbkdf2 -iter 600000 -md sha256 -in $tempZip -out $encPath -pass stdin
    if ($LASTEXITCODE -ne 0) { Fail "OpenSSL encryption failed." }

    # --- Encrypt-then-MAC: HMAC-SHA256 over the ciphertext for integrity. ---
    Write-Host "Writing HMAC-SHA256 integrity tag..." -ForegroundColor Cyan
    $macKey = [Security.Cryptography.SHA256]::Create().ComputeHash(
        [Text.Encoding]::UTF8.GetBytes("rock-os-backup-mac`0" + $plain))
    $hmac = [Security.Cryptography.HMACSHA256]::new($macKey)
    $fs = [IO.File]::OpenRead($encPath)
    try { $mac = $hmac.ComputeHash($fs) } finally { $fs.Dispose() }
    $macHex = ([BitConverter]::ToString($mac) -replace '-', '').ToLower()
    [IO.File]::WriteAllText($macPath, $macHex)
}
finally {
    # CRITICAL: always remove the unencrypted ZIP, even on error.
    if (Test-Path -LiteralPath $tempZip) {
        Remove-Item -LiteralPath $tempZip -Force
        Write-Host "Temporary unencrypted ZIP removed." -ForegroundColor DarkGray
    }
    # Best-effort scrub of the plaintext password from memory.
    $plain = $null
    [GC]::Collect()
}

Write-Host ""
Write-Host "Encrypted backup: $encPath" -ForegroundColor Green
Write-Host "Integrity tag:    $macPath" -ForegroundColor Green
Write-Host "Backup complete." -ForegroundColor Green

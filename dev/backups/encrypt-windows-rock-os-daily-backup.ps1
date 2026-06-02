<#
    encrypt-windows-rock-os-daily-backup.ps1

    WHAT IT DOES
      Creates a FULL, encrypted snapshot of the Rock-OS repo that can fully
      restore the project - including history - on another machine.

      The snapshot includes:
        - Everything Git tracks or sees as new (honors .gitignore, so the
          multi-GB build/test caches like .gotest-cache, .gocache, dist\ and
          *.exe are EXCLUDED), capturing any uncommitted changes.
        - The entire .git folder, preserving commit history, branches, stashes.
        - Any root *.key file - gitignored, but the whole reason this backup
          exists, since git-crypt content is unrecoverable without it.

      The snapshot is zipped (native .NET System.IO.Compression - no external
      zip tool needed), encrypted with OpenSSL AES-256-CBC (pbkdf2, 600k
      iterations, sha256), and an HMAC-SHA256 sidecar (.hmac) is written next to
      it so the decrypt script can detect corruption/tampering - and a wrong
      password - BEFORE decrypting.

    WHERE IT SAVES
      You are prompted to choose a destination each run:
        [1] Local:  %USERPROFILE%\Downloads\Rock-OS-backup
        [2] Drive:  G:\My Drive\Rock-OS\Backups   (Google Drive for Desktop)
      Each backup gets its own dated subfolder (e.g. May-28-2026_3-02PM)
      containing three files: the .enc, its .enc.hmac, and a copy of the decrypt
      script - a self-contained restore bundle. Because the payload is encrypted,
      saving it to a synced Google Drive folder is safe (Drive only sees
      ciphertext). The plaintext ZIP is built in %TEMP% and never lands in a
      synced folder.

    GIT-CRYPT / PROFILES
      The script runs whether or not git-crypt content are locked, and the
      backup is restorable either way (it includes both .git and the .key).
      However, if encrypted content is LOCKED it prints a non-fatal heads-up: it is
      RECOMMENDED to unlock first (START-HERE\Windows\unlock-git-crypt.cmd) so
      encrypted content is captured as readable files rather than git-crypt ciphertext.

    WHAT IT DOES NOT DO
      Does NOT pull, does NOT require a clean working tree, and does NOT modify
      the repo. It snapshots whatever is on disk right now.

    DEPENDENCIES (install these first)
      - Git            (required) - enumerates files and is the thing being
                       backed up.  winget install Git.Git
      - OpenSSL        (required) - does the AES-256-CBC encryption.
                       winget install ShiningLight.OpenSSL.Light
                       (or the Git for Windows "usr\bin\openssl.exe" on PATH)
      - PowerShell 5.1+ or PowerShell 7 (required) - ships with Windows 10/11.
                       The ZIP and HMAC use built-in .NET; no module to install.
      - git-crypt      (only if you unlock encrypted content first, which is recommended).
                       winget install AGWA.git-crypt
      - Google Drive for Desktop (only if you pick destination [2]).
      Make sure git and openssl are on your PATH (run "git --version" and
      "openssl version" in a new terminal to confirm).

    IMPORTANT
      This backup contains the git-crypt .key and (when unlocked) your decrypted
      encrypted content. Handle the .enc - and ESPECIALLY any decrypted output - with
      extreme care. Keep the encryption password somewhere safe; it cannot be
      recovered.
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

$timestamp = Get-Date -Format "MMM-d-yyyy_h-mmtt"

# Each backup gets its own dated folder holding the .enc, its .hmac, and a copy
# of the decrypt script - a self-contained bundle. Let the user pick where it goes.
# The .enc is encrypted, so saving it to a synced Google Drive folder is safe.
$localRoot = Join-Path (Join-Path $env:USERPROFILE "Downloads") "Rock-OS-backup"
$driveRoot = "G:\My Drive\Rock-OS\Backups"

Write-Host ""
Write-Host "Where should this backup be saved?" -ForegroundColor Cyan
Write-Host "  [1] Local Downloads  ($localRoot)" -ForegroundColor Gray
Write-Host "  [2] Google Drive     ($driveRoot)" -ForegroundColor Gray
$choice = Read-Host "Enter 1 or 2 (default 1)"

if ($choice -eq '2') {
    $backupRoot = $driveRoot
    $driveQualifier = (Split-Path -Qualifier $driveRoot) + '\'
    if (-not (Test-Path -LiteralPath $driveQualifier)) {
        Fail "Google Drive ($driveQualifier) is not available. Start Google Drive for Desktop, or re-run and choose local."
    }
} else {
    $backupRoot = $localRoot
}

$backupFolder = Join-Path $backupRoot $timestamp
New-Item -ItemType Directory -Path $backupFolder -Force | Out-Null
$encPath   = Join-Path $backupFolder "rock-os-backup-$timestamp.zip.enc"
$macPath   = "$encPath.hmac"
Write-Host "Saving backup to: $backupFolder" -ForegroundColor Green

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

if (-not (Test-Command git)) { Fail "Git is not installed or not available in PATH." }

# Heads-up (non-fatal): detect whether git-crypt content are still locked.
function Test-EncryptedContentLocked {
    $files = & git -C $repoPath ls-files -- 'Website/ENCRYPTED' 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $files) { return $false }
    foreach ($file in $files) {
        $path = Join-Path $repoPath $file
        if (-not (Test-Path -LiteralPath $path)) { continue }
        $bytes = [IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $path).Path)
        if ($bytes.Length -ge 10 -and [Text.Encoding]::ASCII.GetString($bytes, 1, 8) -eq 'GITCRYPT') {
            return $true
        }
    }
    return $false
}

if (Test-EncryptedContentLocked) {
    Write-Host ""
    Write-Host "NOTE: git-crypt content appear to be LOCKED." -ForegroundColor Yellow
    Write-Host "      The backup will still be fully restorable (it includes .git and your .key)," -ForegroundColor Yellow
    Write-Host "      but it is RECOMMENDED to unlock first so encrypted content is captured as readable" -ForegroundColor Yellow
    Write-Host "      files. Run START-HERE\Windows\unlock-git-crypt.cmd, then re-run this backup." -ForegroundColor Yellow
    Write-Host ""
}

Add-Type -AssemblyName System.IO.Compression.FileSystem

Write-Host "Building file list (tracked + uncommitted, the .git folder, and .key)..." -ForegroundColor Cyan

$repoFull = (Resolve-Path -LiteralPath $repoPath).Path.TrimEnd('\')

# What goes in the snapshot:
#   1. Everything Git tracks OR sees as new, honoring .gitignore. This keeps the
#      multi-GB build/test caches (.gotest-cache, .gocache, dist, *.exe, etc.)
#      OUT, since they are all gitignored, while still capturing uncommitted work.
#   2. The entire .git folder, so history, branches, and stashes are preserved.
#   3. Any root *.key files - gitignored, but the whole reason this backup exists.
$relPaths = [System.Collections.Generic.List[string]]::new()
$seen = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
function Add-Rel($rel) { if ($rel -and $seen.Add($rel)) { $relPaths.Add($rel) } }

$gitFiles = & git -C $repoPath ls-files -c -o --exclude-standard
if ($LASTEXITCODE -ne 0) { Fail "git ls-files failed." }
foreach ($g in $gitFiles) { Add-Rel ($g -replace '/', '\') }

$gitDir = Join-Path $repoFull '.git'
if (Test-Path -LiteralPath $gitDir) {
    foreach ($f in [System.IO.Directory]::EnumerateFiles($gitDir, '*', [System.IO.SearchOption]::AllDirectories)) {
        Add-Rel ($f.Substring($repoFull.Length + 1))
    }
}

foreach ($k in (Get-ChildItem -LiteralPath $repoPath -Filter '*.key' -File -ErrorAction SilentlyContinue)) {
    Add-Rel $k.Name
}

Write-Host "Zipping $($relPaths.Count) files..." -ForegroundColor Cyan

if (Test-Path -LiteralPath $tempZip) { Remove-Item -LiteralPath $tempZip -Force }

# Zip DIRECTLY from the repo - no staging copy - to avoid MAX_PATH doubling.
# Long source paths are read with the \\?\ extended-length prefix.
$skipped = 0
$zip = [System.IO.Compression.ZipFile]::Open($tempZip, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    foreach ($rel in $relPaths) {
        $srcPath = Join-Path $repoFull $rel
        if (-not (Test-Path -LiteralPath $srcPath)) { continue }
        $entryName = $rel -replace '\\', '/'
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

# Copy the decrypt script into the bundle so a restore is self-contained.
$decryptSrc = Join-Path $PSScriptRoot "decrypt-windows-rock-os-daily-backup.ps1"
if (Test-Path -LiteralPath $decryptSrc) {
    Copy-Item -LiteralPath $decryptSrc -Destination (Join-Path $backupFolder "decrypt-windows-rock-os-daily-backup.ps1") -Force
    Write-Host "Decrypt script copied into the backup folder." -ForegroundColor Green
} else {
    Write-Host "WARNING: decrypt script not found at $decryptSrc; not copied." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Backup folder:    $backupFolder" -ForegroundColor Green
Write-Host "Encrypted backup: $encPath" -ForegroundColor Green
Write-Host "Integrity tag:    $macPath" -ForegroundColor Green
Write-Host "Backup complete." -ForegroundColor Green

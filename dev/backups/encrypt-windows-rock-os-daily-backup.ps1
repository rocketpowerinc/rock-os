<#
    windows-rock-os-daily-backup.ps1

    This script creates a ZIP archive of the rock-os repo and encrypts it using OpenSSL AES‑256‑CBC.
    It stores the encrypted backup in the user's Downloads folder.

    IMPORTANT:
    This backup contains the git-crypt .key file. Handle with extreme care.

    NOTE:
    This backup exists because git-crypt encrypted files pushed to GitHub
    are unrecoverable without the key. This script is a fail-safe to ensure
    the key is never lost.
#>

# This script is meant to run from the development directory of Rock-OS
# (e.g. dev\backups\ inside your working clone) so it captures uncommitted
# changes, local branches, and the git-crypt .key file.
# It resolves dev\backups\..\..\  to the repo root.
$repoPath = (Resolve-Path (Join-Path $PSScriptRoot "..\..") -ErrorAction SilentlyContinue).Path
if (-not $repoPath -or -not (Test-Path (Join-Path $repoPath '.git'))) {
    Write-Host "ERROR: This script must be run from inside the Rock-OS development clone (dev\backups\)." -ForegroundColor Red
    Write-Host "       It cannot be run from an arbitrary location." -ForegroundColor Red
    exit 1
}

$backupDir = Join-Path $env:USERPROFILE "Downloads"
$timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
$zipPath = Join-Path $backupDir "rock-os-backup-$timestamp.zip"
$encPath = "$zipPath.enc"
$tempBackupSrc = Join-Path $env:TEMP "rock-os-backup-src-$PID"

function Fail($Message) {
    Write-Host "ERROR: $Message" -ForegroundColor Red
    exit 1
}

function Test-Command($Name) {
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Invoke-Git {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Arguments
    )

    & git -C $repoPath @Arguments
    return $LASTEXITCODE
}

function Test-ProfilesUnlocked {
    $files =
        & git -C $repoPath ls-files -- 'Website/profiles' 2>$null

    if ($LASTEXITCODE -ne 0) {
        Fail "Could not list tracked Profiles files."
    }

    if (-not $files) {
        Write-Host "No tracked Profiles files found. Continuing." -ForegroundColor Yellow
        return $true
    }

    foreach ($file in $files) {
        $path = Join-Path $repoPath $file
        if (-not (Test-Path -LiteralPath $path)) {
            continue
        }

        $bytes = [IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $path))
        if ($bytes.Length -ge 10) {
            $marker = [Text.Encoding]::ASCII.GetString($bytes, 1, 8)
            if ($marker -eq 'GITCRYPT') {
                return $false
            }
        }
    }

    return $true
}

function Assert-RepoReady {
    Write-Host "Checking rock-os repository status..." -ForegroundColor Cyan

    if (-not (Test-Command git)) {
        Fail "Git is not installed or not available in PATH."
    }

    if (-not (Test-Command git-crypt)) {
        Fail "git-crypt is not installed or not available in PATH."
    }

    if (-not (Test-Command openssl)) {
        Fail "OpenSSL is not installed or not available in PATH."
    }

    if (-not (Test-Path $repoPath)) {
        Fail "rock-os folder not found at $repoPath"
    }

    if (-not (Test-Path (Join-Path $repoPath '.git'))) {
        Fail "$repoPath is not a Git clone. GitHub ZIP downloads cannot be safely checked or unlocked."
    }

    Write-Host "Pulling latest repo changes..." -ForegroundColor Cyan
    Invoke-Git pull --ff-only
    if ($LASTEXITCODE -ne 0) {
        Fail "Could not pull latest changes. Commit or clean local changes, then try again."
    }

    $status =
        & git -C $repoPath status --porcelain

    if ($status) {
        Write-Host "Working tree has changes:" -ForegroundColor Yellow
        $status | ForEach-Object { Write-Host "  $_" -ForegroundColor Yellow }
        Fail "Repo must be clean before creating a backup."
    }

    Push-Location $repoPath
    try {
        & git-crypt status *> $null
        if ($LASTEXITCODE -ne 0) {
            Fail "git-crypt status failed. Make sure this repo is initialized for git-crypt."
        }
    }
    finally {
        Pop-Location
    }

    if (-not (Test-ProfilesUnlocked)) {
        Fail "Profiles files are still encrypted. It is useless to run this backup script when files are encrypted. Run START-HERE\Windows\unlock-git-crypt.cmd first, then retry."
    }

    $keyFiles =
        Get-ChildItem -LiteralPath $repoPath -Filter '*.key' -File -ErrorAction SilentlyContinue

    if ($keyFiles.Count -eq 0) {
        Write-Host "WARNING: No .key file was found in the repo root. If your goal is key backup, copy your exported git-crypt key to the repo root first." -ForegroundColor Yellow
    } else {
        Write-Host "git-crypt key file found in repo root. Keep the encrypted backup private." -ForegroundColor Yellow
    }

    Write-Host "Repo is current, clean, and Profiles is decrypted." -ForegroundColor Green
}

Assert-RepoReady

# --- Check repo age ---
$lastWrite = (Get-Item $repoPath).LastWriteTime
$ageDays = (New-TimeSpan -Start $lastWrite -End (Get-Date)).Days

Write-Host "Last modified: $lastWrite ($ageDays days old)" -ForegroundColor Yellow
Write-Host ""

# --- Stage files respecting .gitignore ---
if (Test-Path $tempBackupSrc) { Remove-Item $tempBackupSrc -Recurse -Force }
New-Item -ItemType Directory -Path $tempBackupSrc -Force | Out-Null

Write-Host "Staging files (respecting .gitignore)..." -ForegroundColor Cyan
$gitFiles = & git -C $repoPath ls-files -c -o --exclude-standard
if ($LASTEXITCODE -ne 0) {
    Fail "git ls-files failed."
}

# Convert output to list and explicitly add back any *.key files in root (which are ignored but required)
$filesToCopy = [System.Collections.Generic.List[string]]::new()
foreach ($file in $gitFiles) {
    $filesToCopy.Add($file)
}

$keyFiles = Get-ChildItem -LiteralPath $repoPath -Filter '*.key' -File -ErrorAction SilentlyContinue
foreach ($keyFile in $keyFiles) {
    $relKey = $keyFile.Name
    if (-not $filesToCopy.Contains($relKey)) {
        $filesToCopy.Add($relKey)
    }
}

foreach ($file in $filesToCopy) {
    $srcFile = Join-Path $repoPath $file
    if (-not (Test-Path -LiteralPath $srcFile)) {
        continue
    }
    $destFile = Join-Path $tempBackupSrc $file
    $destDir = Split-Path $destFile -Parent
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }
    Copy-Item -LiteralPath $srcFile -Destination $destFile -Force
}

# --- Wrap operations in try/finally to prevent unencrypted zip leaks ---
try {
    Write-Host "Creating clean ZIP archive..." -ForegroundColor Cyan
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
    Compress-Archive -Path "$tempBackupSrc\*" -DestinationPath $zipPath
    Write-Host "ZIP created at $zipPath" -ForegroundColor Green

    # --- Secure password input with verification ---
    $match = $false
    while (-not $match) {
        $password = Read-Host "Enter encryption password" -AsSecureString
        $confirm = Read-Host "Confirm encryption password" -AsSecureString

        $passwordBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
        $confirmBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($confirm)

        $plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($passwordBstr)
        $plainConfirm = [Runtime.InteropServices.Marshal]::PtrToStringAuto($confirmBstr)

        $match = ($plain -eq $plainConfirm)

        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordBstr)
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($confirmBstr)
        $plainConfirm = $null

        if (-not $match) {
            Write-Host "Passwords do not match. Please try again." -ForegroundColor Yellow
        }
    }

    # --- Encrypt ZIP using OpenSSL ---
    Write-Host "Encrypting ZIP with OpenSSL AES-256-CBC..." -ForegroundColor Cyan
    $passFile = Join-Path $env:TEMP "rock-os-backup-pass-$PID.txt"

    try {
        $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
        [IO.File]::WriteAllText($passFile, $plain, $utf8NoBom)

        # Run OpenSSL with pbkdf2 and explicit sha256 digest
        & openssl enc -aes-256-cbc -salt -pbkdf2 -md sha256 -in $zipPath -out $encPath -pass "file:$passFile"
        if ($LASTEXITCODE -ne 0) {
            Fail "OpenSSL encryption failed."
        }
    }
    finally {
        if (Test-Path -LiteralPath $passFile) {
            # Overwrite temp password file before deleting
            [IO.File]::WriteAllText($passFile, "0000000000000000", $utf8NoBom)
            Remove-Item -LiteralPath $passFile -Force
        }
    }
}
finally {
    # CRITICAL: Always clean up unencrypted files
    if (Test-Path $zipPath) {
        Remove-Item $zipPath -Force
        Write-Host "Temporary unencrypted ZIP removed." -ForegroundColor DarkGray
    }
    if (Test-Path $tempBackupSrc) {
        Remove-Item $tempBackupSrc -Recurse -Force
        Write-Host "Staging area cleaned." -ForegroundColor DarkGray
    }
}

# Clear plaintext password from memory
$plain = $null

Write-Host "Encrypted backup created at $encPath" -ForegroundColor Green
Write-Host ""
Write-Host "Backup complete." -ForegroundColor Green

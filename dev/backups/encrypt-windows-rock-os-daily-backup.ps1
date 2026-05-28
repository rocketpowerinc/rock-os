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

$repoPath = Join-Path $env:USERPROFILE "rock-os"
$backupDir = Join-Path $env:USERPROFILE "Downloads"
$timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
$zipPath = Join-Path $backupDir "rock-os-backup-$timestamp.zip"
$encPath = "$zipPath.enc"

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

Write-Host "Creating ZIP archive..." -ForegroundColor Cyan

# --- Create ZIP ---
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path $repoPath -DestinationPath $zipPath

Write-Host "ZIP created at $zipPath" -ForegroundColor Green

# --- Encrypt ZIP using OpenSSL ---
Write-Host "Encrypting ZIP with OpenSSL AES-256-CBC..." -ForegroundColor Cyan

# Prompt for password
$password = Read-Host "Enter encryption password" -AsSecureString
$passwordBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($passwordBstr)

$passFile =
    Join-Path $env:TEMP "rock-os-backup-pass-$PID.txt"

try {
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [IO.File]::WriteAllText($passFile, $plain, $utf8NoBom)

    # Run OpenSSL without putting the password directly on the command line.
    & openssl enc -aes-256-cbc -salt -pbkdf2 -in $zipPath -out $encPath -pass "file:$passFile"
    if ($LASTEXITCODE -ne 0) {
        Fail "OpenSSL encryption failed."
    }
}
finally {
    if (Test-Path -LiteralPath $passFile) {
        Remove-Item -LiteralPath $passFile -Force
    }
}

# Clear plaintext password from memory
[Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordBstr)
$plain = $null

Write-Host "Encrypted backup created at $encPath" -ForegroundColor Green

# --- Cleanup original ZIP ---
Remove-Item $zipPath -Force
Write-Host "Temporary ZIP removed." -ForegroundColor DarkGray

Write-Host ""
Write-Host "Backup complete." -ForegroundColor Green

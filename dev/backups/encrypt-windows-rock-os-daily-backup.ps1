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

Write-Host "Checking rock-os repository status..." -ForegroundColor Cyan

# --- Check if repo exists ---
if (-not (Test-Path $repoPath)) {
    Write-Host "ERROR: rock-os folder not found at $repoPath" -ForegroundColor Red
    exit 1
}

# --- Check repo age ---
$lastWrite = (Get-Item $repoPath).LastWriteTime
$ageDays = (New-TimeSpan -Start $lastWrite -End (Get-Date)).Days

Write-Host "Last modified: $lastWrite ($ageDays days old)" -ForegroundColor Yellow
Write-Host ""
Write-Host "⚠️  Make sure the repo is UNLOCKED before continuing."
Write-Host "    Open the 'rock-os' icon on your desktop to refresh it."
Write-Host ""

$confirm = Read-Host "Is the repo unlocked and refreshed? (y/n)"
if ($confirm -ne "y") {
    Write-Host "Backup cancelled." -ForegroundColor Red
    exit 1
}

Write-Host "Creating ZIP archive..." -ForegroundColor Cyan

# --- Create ZIP ---
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path $repoPath -DestinationPath $zipPath

Write-Host "ZIP created at $zipPath" -ForegroundColor Green

# --- Encrypt ZIP using OpenSSL ---
Write-Host "Encrypting ZIP with OpenSSL AES-256-CBC..." -ForegroundColor Cyan

# Prompt for password
$password = Read-Host "Enter encryption password" -AsSecureString
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
)

# Run OpenSSL
$opensslCmd = "openssl enc -aes-256-cbc -salt -pbkdf2 -in `"$zipPath`" -out `"$encPath`" -pass pass:$plain"
cmd.exe /c $opensslCmd

# Clear plaintext password from memory
[Runtime.InteropServices.Marshal]::ZeroFreeBSTR(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
)

Write-Host "Encrypted backup created at $encPath" -ForegroundColor Green

# --- Cleanup original ZIP ---
Remove-Item $zipPath -Force
Write-Host "Temporary ZIP removed." -ForegroundColor DarkGray

Write-Host ""
Write-Host "Backup complete." -ForegroundColor Green

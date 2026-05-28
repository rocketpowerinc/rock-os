<#
    decrypt-rock-os-backup.ps1

    Decrypts an OpenSSL AES‑256‑CBC encrypted backup created by
    windows-rock-os-daily-backup.ps1.

    Supports:
      - Drag & drop of .enc file onto the script
      - Manual file selection if run directly

    WARNING:
    The decrypted ZIP contains the git‑crypt .key.
    Handle with extreme care.
#>

param(
    [string]$EncryptedFile
)

# If script was run without drag‑and‑drop, ask for file
if (-not $EncryptedFile) {
    Write-Host "Select the .enc file to decrypt..." -ForegroundColor Cyan
    $EncryptedFile = Read-Host "Path to encrypted file"
}

# Validate file
if (-not (Test-Path $EncryptedFile)) {
    Write-Host "ERROR: File not found: $EncryptedFile" -ForegroundColor Red
    exit 1
}

# Build output ZIP path
$baseName = [System.IO.Path]::GetFileNameWithoutExtension($EncryptedFile)
$dir = [System.IO.Path]::GetDirectoryName($EncryptedFile)
$outZip = Join-Path $dir "$baseName.zip"

Write-Host "Decrypting $EncryptedFile ..." -ForegroundColor Cyan

# Ask for password
$password = Read-Host "Enter decryption password" -AsSecureString
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
)

# Run OpenSSL decrypt
$cmd = "openssl enc -d -aes-256-cbc -pbkdf2 -in `"$EncryptedFile`" -out `"$outZip`" -pass pass:$plain"
cmd.exe /c $cmd

# Clear plaintext password from memory
[Runtime.InteropServices.Marshal]::ZeroFreeBSTR(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
)

Write-Host "Decryption complete." -ForegroundColor Green
Write-Host "Decrypted ZIP saved to: $outZip" -ForegroundColor Yellow
Write-Host ""
Write-Host "⚠️  Reminder: This ZIP contains the git‑crypt key. Protect it." -ForegroundColor Red

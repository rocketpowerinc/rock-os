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

function Fail($Message) {
    Write-Host "ERROR: $Message" -ForegroundColor Red
    exit 1
}

if (-not (Get-Command openssl -ErrorAction SilentlyContinue)) {
    Fail "OpenSSL is not installed or not available in PATH."
}

# If script was run without drag‑and‑drop, ask for file
if (-not $EncryptedFile) {
    Write-Host "Select the .enc file to decrypt..." -ForegroundColor Cyan
    $EncryptedFile = Read-Host "Path to encrypted file"
}

# Validate file
if (-not (Test-Path $EncryptedFile)) {
    Fail "File not found: $EncryptedFile"
}

# Build output ZIP path
$baseName = [System.IO.Path]::GetFileNameWithoutExtension($EncryptedFile)
$dir = [System.IO.Path]::GetDirectoryName($EncryptedFile)
$outZip = Join-Path $dir "$baseName.zip"

Write-Host "Decrypting $EncryptedFile ..." -ForegroundColor Cyan

# Ask for password
$password = Read-Host "Enter decryption password" -AsSecureString
$passwordBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($passwordBstr)

$passFile =
    Join-Path $env:TEMP "rock-os-backup-pass-$PID.txt"

try {
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [IO.File]::WriteAllText($passFile, $plain, $utf8NoBom)

    # Run OpenSSL with pbkdf2 and explicit sha256 digest
    & openssl enc -d -aes-256-cbc -pbkdf2 -md sha256 -in $EncryptedFile -out $outZip -pass "file:$passFile"
    if ($LASTEXITCODE -ne 0) {
        if (Test-Path -LiteralPath $outZip) {
            Remove-Item -LiteralPath $outZip -Force
        }
        Fail "OpenSSL decryption failed. Check the password and encrypted file."
    }
}
finally {
    if (Test-Path -LiteralPath $passFile) {
        # Overwrite temp password file before deleting
        [IO.File]::WriteAllText($passFile, "0000000000000000", $utf8NoBom)
        Remove-Item -LiteralPath $passFile -Force
    }
    # Clear plaintext password from memory
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordBstr)
    $plain = $null
}

Write-Host "Decryption complete." -ForegroundColor Green
Write-Host "Decrypted ZIP saved to: $outZip" -ForegroundColor Yellow
Write-Host ""
Write-Host "Reminder: This ZIP may contain the git-crypt key and decrypted Profiles data. Protect it." -ForegroundColor Red

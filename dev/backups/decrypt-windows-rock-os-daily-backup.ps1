<#
    decrypt-windows-rock-os-daily-backup.ps1

    Decrypts an OpenSSL AES-256-CBC backup created by
    encrypt-windows-rock-os-daily-backup.ps1.

    Before decrypting, it verifies the HMAC-SHA256 sidecar (.hmac) if present,
    so corruption, tampering, or a wrong password is caught up front.

    Supports:
      - Drag & drop of the .enc file onto the script
      - Manual path entry if run directly

    WARNING:
    The decrypted ZIP contains the git-crypt .key and decrypted Profiles.
    Handle with extreme care.
#>

param(
    [string]$EncryptedFile
)

$ErrorActionPreference = 'Stop'

function Fail($Message) {
    Write-Host "ERROR: $Message" -ForegroundColor Red
    exit 1
}

if (-not (Get-Command openssl -ErrorAction SilentlyContinue)) {
    Fail "OpenSSL is not installed or not available in PATH."
}

if (-not $EncryptedFile) {
    Write-Host "Select the .enc file to decrypt..." -ForegroundColor Cyan
    $EncryptedFile = Read-Host "Path to encrypted file"
}

if (-not (Test-Path -LiteralPath $EncryptedFile)) {
    Fail "File not found: $EncryptedFile"
}
$EncryptedFile = (Resolve-Path -LiteralPath $EncryptedFile).Path

# Output path: strip the trailing .enc, then ALWAYS append a fresh timestamp so
# every decryption writes a unique file and never clobbers a previous one.
$baseName = [System.IO.Path]::GetFileNameWithoutExtension($EncryptedFile)
$dir      = [System.IO.Path]::GetDirectoryName($EncryptedFile)
if ($baseName.ToLower().EndsWith('.zip')) {
    $baseName = $baseName.Substring(0, $baseName.Length - 4)
}
$stamp  = Get-Date -Format "MMM-d-yyyy_h-mmtt"
$outZip = Join-Path $dir ("{0}-decrypted-{1}.zip" -f $baseName, $stamp)

Write-Host "Decrypting $EncryptedFile ..." -ForegroundColor Cyan

$password = Read-Host "Enter decryption password" -AsSecureString
$pBstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
$plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($pBstr)
[Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pBstr)

try {
    # --- Verify HMAC-SHA256 before decrypting, if a sidecar exists. ---
    $macPath = "$EncryptedFile.hmac"
    if (Test-Path -LiteralPath $macPath) {
        Write-Host "Verifying HMAC-SHA256 integrity..." -ForegroundColor Cyan
        $expected = (Get-Content -LiteralPath $macPath -Raw).Trim().ToLower()
        $macKey = [Security.Cryptography.SHA256]::Create().ComputeHash(
            [Text.Encoding]::UTF8.GetBytes("rock-os-backup-mac`0" + $plain))
        $hmac = [Security.Cryptography.HMACSHA256]::new($macKey)
        $fs = [IO.File]::OpenRead($EncryptedFile)
        try { $mac = $hmac.ComputeHash($fs) } finally { $fs.Dispose() }
        $actual = ([BitConverter]::ToString($mac) -replace '-', '').ToLower()
        if ($actual -ne $expected) {
            Fail "Integrity check FAILED. Either the password is wrong or the file is corrupted/tampered. Not decrypting."
        }
        Write-Host "Integrity OK." -ForegroundColor Green
    } else {
        Write-Host "No .hmac sidecar found next to this file; skipping integrity check." -ForegroundColor Yellow
    }

    # --- Decrypt. Password is piped via stdin and never written to disk. ---
    $plain | & openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -md sha256 -in $EncryptedFile -out $outZip -pass stdin
    if ($LASTEXITCODE -ne 0) {
        if (Test-Path -LiteralPath $outZip) { Remove-Item -LiteralPath $outZip -Force }
        Fail "OpenSSL decryption failed. Check the password and encrypted file."
    }
}
finally {
    $plain = $null
    [GC]::Collect()
}

Write-Host "Decryption complete." -ForegroundColor Green
Write-Host "Decrypted ZIP saved to: $outZip" -ForegroundColor Yellow
Write-Host ""
Write-Host "Reminder: This ZIP contains the git-crypt key and decrypted Profiles. Protect it." -ForegroundColor Red

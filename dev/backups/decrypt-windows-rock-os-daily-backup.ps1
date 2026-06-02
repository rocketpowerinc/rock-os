<#
    decrypt-windows-rock-os-daily-backup.ps1

    WHAT IT DOES
      Decrypts a backup created by encrypt-windows-rock-os-daily-backup.ps1
      (OpenSSL AES-256-CBC, pbkdf2, 600k iterations, sha256) back into a .zip.
      Unzip the result to get a full copy of the repo, including .git and the
      git-crypt .key.

    INTEGRITY CHECK
      Before decrypting, it verifies the HMAC-SHA256 sidecar that sits next to
      the .enc. The sidecar path is just "<your .enc>.hmac", so the .enc and its
      .enc.hmac must be in the SAME folder with matching names. If the sidecar
      is present and valid, decryption proceeds; if it does not match, the script
      refuses to decrypt (wrong password, or corrupted/tampered file). If no
      .hmac is found, it prints a warning and decrypts anyway - the .enc alone is
      enough to recover the data, you just lose the early integrity check.

    INPUT
      - Drag & drop the .enc file onto the script (it strips the quotes Windows
        adds), or
      - Run it directly and paste/type the path when prompted.

    OUTPUT
      Writes "<name>-decrypted-<MMM-d-yyyy_h-mmtt>.zip" next to the .enc. The
      timestamp guarantees each run is a unique file and never overwrites a
      previous decryption.

    DEPENDENCIES (install these first)
      - OpenSSL        (required) - does the decryption.
                       winget install ShiningLight.OpenSSL.Light
                       (or the Git for Windows "usr\bin\openssl.exe" on PATH)
      - PowerShell 5.1+ or PowerShell 7 (required) - ships with Windows 10/11.
                       The HMAC check uses built-in .NET; no module to install.
      - git-crypt      (only AFTER restore, if the backed-up encrypted content were
                       locked - run "git-crypt unlock" with the included .key to
                       make them readable).  winget install AGWA.git-crypt
      Confirm OpenSSL is on your PATH with "openssl version" in a new terminal.

    WARNING
      The decrypted ZIP contains the git-crypt .key and (if the backup was made
      while unlocked) your decrypted encrypted content. Handle with extreme care and
      delete it once you have restored what you need.
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
    Write-Host "Tip: you can drag and drop the .enc file onto this window, then press Enter." -ForegroundColor Cyan
    $EncryptedFile = Read-Host "Path to encrypted (.enc) file"
}

# Drag-and-drop often wraps the path in quotes - strip them.
$EncryptedFile = $EncryptedFile.Trim().Trim('"')

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
Write-Host "Reminder: This ZIP contains the git-crypt key and decrypted encrypted content. Protect it." -ForegroundColor Red

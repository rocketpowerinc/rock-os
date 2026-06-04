# GO-tests.ps1
# Builds and tests the Rock-OS Go server, printing results to the console AND
# writing the combined output to dev/tests/latest-go-test-results.txt. Each run
# overwrites that single file; the run's timestamp is the first line inside it.
# This does NOT build release binaries or change any version - it only runs
# `go build`, `go vet`, and `go test`.

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path     # dev/tests
$RepoRoot  = Split-Path -Parent (Split-Path -Parent $ScriptDir)  # repo root
$ServerDir = Join-Path $RepoRoot 'cmd/rock-os'
$LogFile   = Join-Path $ScriptDir 'latest-go-test-results.txt'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host '[ERROR] go command not found. Install Go and add it to your PATH.' -ForegroundColor Red
    exit 1
}

Push-Location $ServerDir
try {
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add("Rock-OS Go server tests - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
    $lines.Add("$(go version)")

    $lines.Add("`n=== go build ./... ===")
    $buildOut = @(& go build ./... 2>&1 | ForEach-Object { "$_" })
    $buildOk = ($LASTEXITCODE -eq 0)
    if ($buildOut.Count) { $lines.AddRange([string[]]$buildOut) }
    $lines.Add("(exit code: $LASTEXITCODE)")

    $lines.Add("`n=== go vet ./... ===")
    $vetOut = @(& go vet ./... 2>&1 | ForEach-Object { "$_" })
    $vetOk = ($LASTEXITCODE -eq 0)
    if ($vetOut.Count) { $lines.AddRange([string[]]$vetOut) }
    $lines.Add("(exit code: $LASTEXITCODE)")

    $lines.Add("`n=== go test ./... ===")
    $testOut = @(& go test ./... 2>&1 | ForEach-Object { "$_" })
    $testOk = ($LASTEXITCODE -eq 0)
    if ($testOut.Count) { $lines.AddRange([string[]]$testOut) }
    $lines.Add("(exit code: $LASTEXITCODE)")

    $summary = if ($buildOk -and $vetOk -and $testOk) { 'RESULT: PASS' } else { 'RESULT: FAIL' }
    $lines.Add("`n$summary")
}
finally {
    Pop-Location
}

# Console output
$lines | ForEach-Object { Write-Host $_ }

# File output (UTF-8 without BOM, with project-standard LF line endings)
$logText = ($lines -join "`n") + "`n"
[System.IO.File]::WriteAllText($LogFile, $logText, [System.Text.UTF8Encoding]::new($false))

Write-Host ''
Write-Host "Full results written to: $LogFile" -ForegroundColor Green

if (-not ($buildOk -and $vetOk -and $testOk)) {
    exit 1
}

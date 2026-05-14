# ROCKOS LIVE SERVER (Unified PowerShell Script)

# Change to script directory
Set-Location -Path $PSScriptRoot

# Check for static-web-server.exe in the static-web-server subfolder
$serverExe = Join-Path $PSScriptRoot 'static-web-server/static-web-server.exe'
if (-not (Test-Path $serverExe)) {
  Write-Host "static-web-server.exe missing in static-web-server/ folder."
  Pause
  exit
}

# Prompt the user to enter their local IP address
Write-Host "[DEBUG] Awaiting user input for local IP..."

$localIP = Read-Host "Enter your local IP address (e.g., 192.168.1.2) or leave blank for 127.0.0.1"

Write-Host "[DEBUG] User entered: '$localIP'"

if ([string]::IsNullOrWhiteSpace($localIP)) {
  $localIP = "127.0.0.1"
  Write-Host "[DEBUG] Defaulted to: '$localIP'"
}

# Build URL safely
$url = "http://${localIP}:8000"

Write-Host "Opening browser at $url"

# Open browser
try {
  Start-Process $url
}
catch {
  Write-Host "Could not open browser automatically."
  Write-Host "Please open $url manually."
}

# Start static-web-server.exe in the background
$server = Start-Process `
  -PassThru `
  -NoNewWindow `
  -FilePath $serverExe `
  -ArgumentList "--host 0.0.0.0 --port 8000 --root ."

Write-Host "Started static-web-server.exe with PID $($server.Id) from $serverExe"

# Start the markdown index generator as a background job
$indexJob = Start-Job -ScriptBlock {

  param($scriptRoot)

  Set-Location $scriptRoot

  while ($true) {

    $root = "markdown"

    if (Test-Path $root) {

      $files = Get-ChildItem $root -Recurse -Filter *.md | ForEach-Object {

        $_.FullName.Replace((Get-Location).Path + "\", "") -replace "\\", "/"

      }

      @($files) | ConvertTo-Json | Set-Content "markdown-index.json"

      Write-Host "Updated markdown-index.json"
    }
    else {
      Write-Host "markdown folder not found."
    }

    Start-Sleep -Seconds 2
  }
} -ArgumentList $PSScriptRoot

Write-Host "Started index generator as Job Id $($indexJob.Id)"

# Cleanup function
function Cleanup {

  Write-Host ""
  Write-Host "Stopping server and index generator..."

  # Stop web server
  if ($server -and -not $server.HasExited) {

    Stop-Process -Id $server.Id -Force

    Write-Host "Stopped static-web-server.exe (PID $($server.Id))"
  }

  # Stop background job
  if ($indexJob -and $indexJob.State -eq "Running") {

    Stop-Job $indexJob | Out-Null
    Remove-Job $indexJob | Out-Null

    Write-Host "Stopped index generator job (Id $($indexJob.Id))"
  }

  Pause
  exit
}

# Register cleanup on PowerShell exit
Register-EngineEvent PowerShell.Exiting -Action {
  Cleanup
} | Out-Null

# Wait until the server exits
try {
  Wait-Process -Id $server.Id
}
finally {
  Cleanup
}
# ROCKOS LIVE SERVER

Set-Location -Path $PSScriptRoot

$serverExe = Join-Path `
  $PSScriptRoot `
  'static-web-server/static-web-server.exe'

if (-not (Test-Path $serverExe)) {

  Write-Host ""
  Write-Host "static-web-server.exe missing."
  Write-Host ""

  Pause
  exit
}

Write-Host ""
Write-Host "[ROCKOS LIVE SERVER]"
Write-Host ""

$localIP = Read-Host `
  "Enter local IP (leave blank for 127.0.0.1)"

if ([string]::IsNullOrWhiteSpace($localIP)) {

  $localIP = "127.0.0.1"
}

$url = "http://${localIP}:8000"

Write-Host ""
Write-Host "Opening browser at:"
Write-Host $url
Write-Host ""

try {

  Start-Process $url
}
catch {

  Write-Host "Could not open browser automatically."
}

# Start web server
$server = Start-Process `
  -FilePath $serverExe `
  -ArgumentList "--host 0.0.0.0 --port 8000 --root ." `
  -PassThru `
  -NoNewWindow

Write-Host "Started static-web-server.exe"
Write-Host "PID: $($server.Id)"
Write-Host ""

# Markdown watcher
$indexJob = Start-Job `
  -ArgumentList $PSScriptRoot `
  -ScriptBlock {

  param($scriptRoot)

  Set-Location $scriptRoot

  $root = Join-Path $scriptRoot "markdown"

  $jsonFile = Join-Path `
    $scriptRoot `
    "markdown-index.json"

  # Ensure file exists immediately
  if (-not (Test-Path $jsonFile)) {

    [System.IO.File]::WriteAllText(
      $jsonFile,
      "[]",
      [System.Text.UTF8Encoding]::new($false)
    )
  }

  while ($true) {

    try {

      if (Test-Path $root) {

        # FORCE ARRAY
        $files = @(Get-ChildItem `
            -Path $root `
            -Recurse `
            -Filter *.md `
            -File |
          ForEach-Object {

            $_.FullName.Replace(
              $scriptRoot + "\",
              ""
            ) -replace "\\", "/"
          })

        # STABLE JSON
        $json =
        [System.Text.Json.JsonSerializer]::Serialize($files)

        # DIRECT OVERWRITE
        [System.IO.File]::WriteAllText(
          $jsonFile,
          $json,
          [System.Text.UTF8Encoding]::new($false)
        )

        Write-Host "Updated markdown-index.json"
      }
      else {

        [System.IO.File]::WriteAllText(
          $jsonFile,
          "[]",
          [System.Text.UTF8Encoding]::new($false)
        )
      }
    }
    catch {

      Write-Host ""
      Write-Host "[INDEX ERROR]"
      Write-Host $_
      Write-Host ""
    }

    Start-Sleep -Seconds 2
  }
}

Write-Host "Started markdown watcher"
Write-Host "Job ID: $($indexJob.Id)"
Write-Host ""

try {

  Wait-Process -Id $server.Id
}
finally {

  Write-Host ""
  Write-Host "Stopping services..."
  Write-Host ""

  # Stop web server
  if ($server -and -not $server.HasExited) {

    Stop-Process `
      -Id $server.Id `
      -Force `
      -ErrorAction SilentlyContinue

    Write-Host "Stopped static-web-server.exe"
  }

  # Stop watcher job
  if ($indexJob) {

    Stop-Job `
      $indexJob `
      -ErrorAction SilentlyContinue | Out-Null

    Remove-Job `
      $indexJob `
      -Force `
      -ErrorAction SilentlyContinue | Out-Null

    Write-Host "Stopped markdown watcher"
  }

  Write-Host ""
  Pause
}
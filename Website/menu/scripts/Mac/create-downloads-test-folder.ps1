$Target = Join-Path $HOME 'Downloads/Rock-OS-Script-Test-PowerShell'

Write-Host 'This PowerShell test script creates this folder:'
Write-Host $Target

New-Item -ItemType Directory -Path $Target -Force | Out-Null

Write-Host 'Folder is ready:'
Write-Host $Target

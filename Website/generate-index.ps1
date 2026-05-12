while ($true) {

$root = "markdown"

$files = Get-ChildItem $root -Recurse -Filter *.md |
ForEach-Object {
    $_.FullName.Replace((Get-Location).Path + "\", "") -replace "\\","/"
}

$files | ConvertTo-Json | Set-Content "markdown-index.json"

Write-Host "Updated markdown-index.json"

Start-Sleep -Seconds 2

}

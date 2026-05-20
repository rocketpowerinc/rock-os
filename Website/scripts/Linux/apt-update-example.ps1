Write-Host 'This example shows how to run apt update from the web terminal.'
Write-Host 'Use the Hide input checkbox before sending your sudo password.'
Write-Host 'This script uses sudo -S so sudo reads the password from the dashboard input.'
Write-Host ''

sudo -S -p 'sudo password: ' apt update

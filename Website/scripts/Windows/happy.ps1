# Ask the user if they are happy
$answer = Read-Host "Are you happy? (yes or no)"

# Convert input to lowercase for easier comparison
$answer = $answer.ToLower()

if ($answer -eq "yes") {
  Write-Host "Great!"
}
elseif ($answer -eq "no") {
  Write-Host "Why are you sad?"
}
else {
  Write-Host "Please answer with yes or no."
}

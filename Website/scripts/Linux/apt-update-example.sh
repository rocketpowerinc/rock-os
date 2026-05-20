#!/usr/bin/env sh
set -eu

printf '%s\n' "This example shows how to run apt update from the web terminal."
printf '%s\n' "Use the Hide input checkbox before sending your sudo password."
printf '%s\n' "This script uses sudo -S so sudo reads the password from the dashboard input."
printf '\n'

sudo -S -p "sudo password: " apt update

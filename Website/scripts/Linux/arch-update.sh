#!/usr/bin/env sh
set -eu

printf '%s\n' "This example shows how to update an Arch Linux system from the web terminal."
printf '%s\n' "Use the Hide input checkbox before sending your sudo password."
printf '%s\n' "This script uses sudo -S so sudo reads the password from the dashboard input."
printf '%s\n' "Arch does not split update and upgrade like Debian/Ubuntu; pacman -Syu does both."
printf '\n'

if ! command -v pacman >/dev/null 2>&1; then
    printf '%s\n' "pacman was not found. This script is intended for Arch-based systems."
    exit 1
fi

sudo -S -p "sudo password: " pacman -Syu

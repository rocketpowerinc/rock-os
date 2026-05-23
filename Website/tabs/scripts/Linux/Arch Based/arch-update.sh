#!/usr/bin/env sh
set -eu

printf '%s\n' "This example shows how to update an Arch Linux system."
printf '%s\n' "Rock-OS opens this script in your OS terminal so sudo prompts work normally."
printf '%s\n' "Arch does not split update and upgrade like Debian/Ubuntu; pacman -Syu does both."
printf '\n'

if ! command -v pacman >/dev/null 2>&1; then
    printf '%s\n' "pacman was not found. This script is intended for Arch-based systems."
    exit 1
fi

sudo pacman -Syu

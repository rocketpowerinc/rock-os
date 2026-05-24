#!/usr/bin/env bash
set -eu

if ! command -v pacman >/dev/null 2>&1; then
    printf '%s\n' "pacman was not found. This script is intended for Arch-based systems."
    exit 1
fi

sudo pacman -Syu

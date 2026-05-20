#!/usr/bin/env sh
set -eu

target="$HOME/Downloads/Rock-OS-Script-Test"

printf '%s\n' "This test script creates this folder:"
printf '%s\n\n' "$target"

mkdir -p "$target"
printf '%s\n' "Folder is ready:"
printf '%s\n' "$target"

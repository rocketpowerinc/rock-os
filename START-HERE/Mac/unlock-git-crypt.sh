#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

if ! command -v git-crypt >/dev/null 2>&1; then
    echo "git-crypt was not found."
    echo "Install git-crypt, then run this script again."
    exit 1
fi

set -- ./*.key

if [ "$1" = "./*.key" ]; then
    echo "No .key file was found in the repo root."
    echo "Copy your exported git-crypt key here, then run this script again."
    exit 1
fi

if [ "$#" -gt 1 ]; then
    echo "More than one .key file was found in the repo root."
    echo "Keep only the git-crypt key here, then run this script again."
    exit 1
fi

echo "Unlocking repository with $1..."
key_name="$(basename "$1")"
temp_key="${TMPDIR:-/tmp}/rock-os-git-crypt-$$.key"
cp "$1" "$temp_key"
rm "$1"
set +e
git-crypt unlock "$temp_key"
unlock_result=$?
set -e
if ! cp "$temp_key" "./$key_name"; then
    echo "Failed to copy the key back to the repo root."
    echo "Your key is still at $temp_key."
    exit 1
fi
rm -f "$temp_key"
if [ "$unlock_result" -ne 0 ]; then
    echo "Failed to unlock the repository."
    exit "$unlock_result"
fi

echo "Repository unlocked."
echo "Key restored to ./$key_name."

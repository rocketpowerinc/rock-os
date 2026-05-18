#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")"

if ! command -v git-crypt >/dev/null 2>&1; then
    echo "git-crypt was not found."
    echo "Install git-crypt, then run this script again."
    exit 1
fi

if [ ! -d ".git" ]; then
    echo "This script must be run from the Rock-OS repo root."
    exit 1
fi

echo "Locking private markdown with git-crypt..."
if ! git-crypt lock; then
    echo "Failed to lock the repository."
    echo "Close open private files or commit/stash changes, then try again."
    exit 1
fi

echo "Repository locked."

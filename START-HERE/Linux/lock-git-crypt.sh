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

if [ ! -d ".git" ]; then
    echo "This script must be run from the Rock-OS repo root."
    exit 1
fi

echo "Locking Profiles with git-crypt..."
if ! git-crypt lock; then
    echo "Failed to lock the repository."
    echo "Close open Profiles files or commit/stash changes, then try again."
    exit 1
fi

echo "Repository locked."

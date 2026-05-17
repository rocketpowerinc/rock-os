#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")"

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
git-crypt unlock "$1"
echo "Repository unlocked."

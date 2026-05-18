#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/Website"

if [ -x "./rock-os-wiki-v1.0-linux-amd64" ]; then
    echo "Starting Rock-OS from release binary..."
    ./rock-os-wiki-v1.0-linux-amd64 --host local
elif [ -f "./rock-os-wiki-v1.0-linux-amd64" ]; then
    echo "Release binary found but is not executable. Making it executable..."
    chmod +x ./rock-os-wiki-v1.0-linux-amd64
    ./rock-os-wiki-v1.0-linux-amd64 --host local
else
    echo "Release binary not found. Starting Rock-OS from Go source..."
    go run . --host local
fi

#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/Website"

BINARY=""

if [ -f "./rock-os-wiki-linux-amd64" ]; then
    BINARY="./rock-os-wiki-linux-amd64"
else
    for FILE in $(find . -maxdepth 1 -type f -name 'rock-os-wiki-v*-linux-amd64' | sort -r); do
        BINARY="$FILE"
        break
    done
fi

if [ -n "$BINARY" ] && [ -x "$BINARY" ]; then
    echo "Starting Rock-OS from release binary..."
    "$BINARY" --host local
elif [ -n "$BINARY" ] && [ -f "$BINARY" ]; then
    echo "Release binary found but is not executable. Making it executable..."
    chmod +x "$BINARY"
    "$BINARY" --host local
else
    echo "Release binary not found. Starting Rock-OS from Go source..."
    go run . --host local
fi

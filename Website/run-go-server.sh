#!/usr/bin/env sh
set -eu

# Source-only launcher for development or troubleshooting when you want to run
# the Go server from local source instead of using or downloading a release
# binary. It builds a local dev binary first so behavior matches Windows.

cd "$(dirname "$0")"

DEV_BINARY="./rock-os-wiki-dev"

echo "Building Rock-OS from Go source..."
GOCACHE="$PWD/.gocache" go build -o "$DEV_BINARY" .

echo "Starting Rock-OS from local dev binary..."
"$DEV_BINARY" --host local

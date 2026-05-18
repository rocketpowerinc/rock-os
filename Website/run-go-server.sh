#!/usr/bin/env sh
set -eu

# Source-only launcher for development or troubleshooting when you want to run
# the Go server directly instead of using or downloading a release binary.

cd "$(dirname "$0")"

echo "Starting Rock-OS from Go source..."
GOCACHE="$PWD/.gocache" go run main.go --host local

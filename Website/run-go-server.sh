#!/usr/bin/env sh
set -eu

# Source-only launcher for development or troubleshooting when you want to run
# the Go server from local source instead of using or downloading a release
# binary. It builds a local dev binary first so behavior matches Windows.

cd "$(dirname "$0")"

ROCK_OS_HOST="${ROCK_OS_HOST:-127.0.0.1}"
case "${1:-}" in
    lan|local|all)
        ROCK_OS_HOST="local"
        ;;
    127.0.0.1|0.0.0.0)
        ROCK_OS_HOST="$1"
        ;;
    "")
        ;;
    *)
        ROCK_OS_HOST="$1"
        ;;
esac

DEV_BINARY="./rock-os-wiki-dev"
WEBSITE_DIR="$PWD"
SOURCE_DIR="$(CDPATH= cd -- ".." && pwd)/cmd/rock-os-wiki"

echo "Building Rock-OS from Go source..."
cd "$SOURCE_DIR"
GOCACHE="$WEBSITE_DIR/.gocache" go build -o "$WEBSITE_DIR/$DEV_BINARY" .

echo "Starting Rock-OS from local dev binary..."
cd "$WEBSITE_DIR"
"$DEV_BINARY" --site-root "$WEBSITE_DIR" --host "$ROCK_OS_HOST"

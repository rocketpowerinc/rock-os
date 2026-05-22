#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

case "$(uname -s)" in
    Darwin)
        exec sh "$SCRIPT_DIR/Mac/stop-rock-os.sh" "$@"
        ;;
    *)
        exec sh "$SCRIPT_DIR/Linux/stop-rock-os.sh" "$@"
        ;;
esac

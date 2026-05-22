#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

case "$(uname -s)" in
    Darwin)
        exec sh "$SCRIPT_DIR/Mac/start-rock-os.sh" "$@"
        ;;
    *)
        exec sh "$SCRIPT_DIR/Linux/start-rock-os.sh" "$@"
        ;;
esac

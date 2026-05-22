#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

case "$(uname -s)" in
    Darwin)
        exec sh "$SCRIPT_DIR/Mac/repo-status.sh" "$@"
        ;;
    *)
        exec sh "$SCRIPT_DIR/Linux/repo-status.sh" "$@"
        ;;
esac

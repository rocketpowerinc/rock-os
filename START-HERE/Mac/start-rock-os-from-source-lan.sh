#!/usr/bin/env sh
set -eu

# Source-only LAN launcher. This intentionally exposes Rock-OS to trusted
# devices on your local network while still building from local Go source.

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
exec sh "$SCRIPT_DIR/start-rock-os-from-source.sh" lan

#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

case "$(uname -s)" in
    Darwin)
        LOCAL_INSTALLER="$SCRIPT_DIR/Mac/install-rock-os.sh"
        REMOTE_INSTALLER="https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Mac/install-rock-os.sh"
        ;;
    *)
        LOCAL_INSTALLER="$SCRIPT_DIR/Linux/install-rock-os.sh"
        REMOTE_INSTALLER="https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Linux/install-rock-os.sh"
        ;;
esac

if [ -f "$LOCAL_INSTALLER" ]; then
    exec sh "$LOCAL_INSTALLER" "$@"
fi

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$REMOTE_INSTALLER" | sh
elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$REMOTE_INSTALLER" | sh
else
    echo "curl or wget is required to fetch the Rock-OS installer."
    exit 1
fi

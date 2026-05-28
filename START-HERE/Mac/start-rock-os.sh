#!/usr/bin/env sh
set -eu

green() {
    printf '\033[32m%s\033[0m\n' "$1"
}

yellow() {
    printf '\033[33m%s\033[0m\n' "$1"
}

red() {
    printf '\033[31m%s\033[0m\n' "$1"
}

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
SELF_SCRIPT="$SCRIPT_DIR/$(basename -- "$0")"
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

if [ ! -e "$REPO_ROOT/.git" ]; then
    red "This folder is not a cloned Git repository."
    yellow "GitHub ZIP downloads do not include the .git folder, so git-crypt cannot unlock Profiles."
    yellow "Use this instead:"
    printf '%s\n' "git clone https://github.com/rocketpowerinc/rock-os.git"
    printf '%s\n' "cd rock-os"
    printf '%s\n' "cd START-HERE/Mac"
    exit 1
fi

pull_updates() {
    if ! command -v git >/dev/null 2>&1; then
        yellow "Git is not installed. Skipping repo update and using local files."
        return
    fi

    green "Checking for Rock-OS repo updates..."
    before_head="$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || true)"
    if git -C "$REPO_ROOT" pull --ff-only; then
        after_head="$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || true)"
        green "Rock-OS repo is up to date."
        if [ -n "$before_head" ] && [ -n "$after_head" ] && [ "$before_head" != "$after_head" ] && [ "${ROCK_OS_RESTARTED_AFTER_PULL:-}" != "1" ]; then
            return 222
        fi
    else
        yellow "Could not update from GitHub. Continuing with local files."
        yellow "If you have local changes, commit them before pulling updates."
    fi
}

check_go() {
    if command -v go >/dev/null 2>&1; then
        green "Go installed. Source fallback available."
    else
        yellow "Go is not installed. Not needed while using a release binary."
    fi
}

read_version_file() {
    if [ -f "$VERSION_FILE" ]; then
        sed -n '/^[[:space:]]*#/d; /^[[:space:]]*$/d; s/^[[:space:]]*//; s/[[:space:]]*$//; p; q' "$VERSION_FILE"
    fi
}

write_version_file() {
    {
        printf '%s\n' "$1"
        printf '%s\n' "# Local Rock-OS release marker used by START-HERE launchers."
        printf '%s\n' "# First non-comment line is the downloaded release tag."
        printf '%s\n' "# If this tag differs from GitHub's latest release, the launcher downloads a fresh binary."
    } > "$VERSION_FILE"
}

if pull_updates; then
    :
else
    status="$?"
    if [ "$status" -eq 222 ]; then
        yellow "Launcher files changed during update. Restarting Rock-OS launcher once..."
        ROCK_OS_RESTARTED_AFTER_PULL=1 exec "$SELF_SCRIPT" "$@"
    fi
    exit "$status"
fi

cd "$REPO_ROOT/Website"

REPO="rocketpowerinc/rock-os"
VERSION_FILE=".rock-os-version"
BINARY=""

case "$(uname -s)" in
    Darwin)
        PLATFORM="macos"
        ;;
    Linux)
        PLATFORM="linux"
        ;;
    *)
        PLATFORM="linux"
        ;;
esac

case "$(uname -m)" in
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        ARCH="amd64"
        ;;
esac

STABLE_ASSET="rock-os-$PLATFORM-$ARCH"

check_go

latest_tag=""
if command -v curl >/dev/null 2>&1; then
    latest_tag="$(curl -fsSL -H "User-Agent: rock-os-start-script" "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1 || true)"
elif command -v wget >/dev/null 2>&1; then
    latest_tag="$(wget -qO- --header="User-Agent: rock-os-start-script" "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1 || true)"
fi

if [ -n "$latest_tag" ]; then
    local_tag=""
    if [ -f "$VERSION_FILE" ]; then
        local_tag="$(read_version_file)"
    fi

    if [ ! -f "./$STABLE_ASSET" ] || [ "$local_tag" != "$latest_tag" ]; then
        yellow "Downloading Rock-OS $latest_tag for $PLATFORM $ARCH..."
        VERSIONED_ASSET="rock-os-$latest_tag-$PLATFORM-$ARCH"
        downloaded=""

        for ASSET in "$STABLE_ASSET" "$VERSIONED_ASSET"; do
            temp_asset="./$ASSET.download"
            rm -f "$temp_asset"

            if command -v curl >/dev/null 2>&1; then
                curl -fL -H "User-Agent: rock-os-start-script" -o "$temp_asset" "https://github.com/$REPO/releases/latest/download/$ASSET" || true
            elif command -v wget >/dev/null 2>&1; then
                wget -q --header="User-Agent: rock-os-start-script" -O "$temp_asset" "https://github.com/$REPO/releases/latest/download/$ASSET" || true
            fi

            if [ -s "$temp_asset" ]; then
                mv "$temp_asset" "./$STABLE_ASSET"
                chmod +x "./$STABLE_ASSET"
                write_version_file "$latest_tag"
                downloaded="true"
                green "Downloaded Rock-OS $latest_tag."
                break
            fi

            rm -f "$temp_asset"
        done

        if [ -z "$downloaded" ]; then
            yellow "Could not download the latest Rock-OS binary. Continuing with local files..."
        fi
    else
        green "Rock-OS binary is current ($latest_tag)."
    fi
else
    yellow "Could not check the latest Rock-OS release. Continuing with local files..."
fi

if [ -f "./$STABLE_ASSET" ]; then
    BINARY="./$STABLE_ASSET"
else
    for FILE in $(find . -maxdepth 1 -type f -name "rock-os-v*-$PLATFORM-$ARCH" | sort -r); do
        BINARY="$FILE"
        break
    done
fi

if [ -n "$BINARY" ] && [ -x "$BINARY" ]; then
    green "Starting Rock-OS..."
    "$BINARY" --host "$ROCK_OS_HOST"
elif [ -n "$BINARY" ] && [ -f "$BINARY" ]; then
    yellow "Release binary found but is not executable. Making it executable..."
    chmod +x "$BINARY"
    green "Starting Rock-OS..."
    "$BINARY" --host "$ROCK_OS_HOST"
else
    yellow "Release binary not found. Starting Rock-OS from Go source..."
    if ! command -v go >/dev/null 2>&1; then
        red "Cannot start from source because Go is not installed."
        exit 1
    fi
    WEBSITE_DIR="$PWD"
    cd "$REPO_ROOT/cmd/rock-os"
    GOCACHE="$WEBSITE_DIR/.gocache" go run . --site-root "$WEBSITE_DIR" --host "$ROCK_OS_HOST"
fi

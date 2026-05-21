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

if [ ! -e "$SCRIPT_DIR/.git" ]; then
    red "This folder is not a cloned Git repository."
    yellow "GitHub ZIP downloads do not include the .git folder, so git-crypt cannot unlock Private markdown."
    yellow "Use this instead:"
    printf '%s\n' "git clone https://github.com/rocketpowerinc/rock-os.git"
    printf '%s\n' "cd rock-os"
    exit 1
fi

pull_updates() {
    if ! command -v git >/dev/null 2>&1; then
        yellow "Git is not installed. Skipping repo update and using local files."
        return
    fi

    green "Checking for Rock-OS repo updates..."
    if git -C "$SCRIPT_DIR" pull --ff-only >/dev/null; then
        green "Rock-OS repo is up to date."
    else
        yellow "Could not update from GitHub. Continuing with local files."
        yellow "If you have local changes, commit them before pulling updates."
    fi
}

check_git_crypt() {
    if command -v git-crypt >/dev/null 2>&1; then
        green "git-crypt is installed."
    else
        red "git-crypt is not installed. Install git-crypt before unlocking Private markdown."
    fi
}

check_go() {
    if command -v go >/dev/null 2>&1; then
        green "Go is installed."
    elif [ -n "$BINARY" ]; then
        yellow "Go is not installed. Not needed while using a release binary."
    else
        red "Go is not installed. Install Go from https://go.dev/dl/ before using source fallback."
    fi
}

check_private() {
    locked_marker="${TMPDIR:-/tmp}/rock-os-private-locked-$$"
    found_marker="${TMPDIR:-/tmp}/rock-os-private-found-$$"
    rm -f "$locked_marker"
    rm -f "$found_marker"

    if [ -d "markdown/Private" ]; then
        find "markdown/Private" -type f | while IFS= read -r file; do
            printf 'found' > "$found_marker"

            if dd if="$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
                printf 'locked' > "$locked_marker"
            fi
        done
    fi

    private_files="$(git -C .. ls-files -- 'Website/markdown/Private' 2>/dev/null || true)"
    printf '%s\n' "$private_files" | while IFS= read -r file; do
        [ -n "$file" ] || continue
        [ -f "../$file" ] || continue
        printf 'found' > "$found_marker"

        if dd if="../$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
            printf 'locked' > "$locked_marker"
        fi
    done

    if [ -f "$locked_marker" ]; then
        rm -f "$locked_marker"
        rm -f "$found_marker"
        red "Private Markdown Folder Locked."
    else
        rm -f "$found_marker"
        green "Private Markdown Folder Unlocked."
    fi
}

pull_updates

cd "$SCRIPT_DIR/Website"

REPO="rocketpowerinc/rock-os"
VERSION_FILE=".rock-os-wiki-version"
BINARY=""
BINARY_SOURCE=""

green "[Rock-OS] Launcher online."
if [ "$ROCK_OS_HOST" = "127.0.0.1" ]; then
    green "Host mode: local-only. Other computers cannot connect."
else
    yellow "Host mode: LAN. Use only on a trusted network."
fi

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

STABLE_ASSET="rock-os-wiki-$PLATFORM-$ARCH"
green "Detected $PLATFORM $ARCH."

latest_tag=""
if command -v curl >/dev/null 2>&1; then
    latest_tag="$(curl -fsSL -H "User-Agent: rock-os-start-script" "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1 || true)"
elif command -v wget >/dev/null 2>&1; then
    latest_tag="$(wget -qO- --header="User-Agent: rock-os-start-script" "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1 || true)"
fi

if [ -n "$latest_tag" ]; then
    local_tag=""
    if [ -f "$VERSION_FILE" ]; then
        local_tag="$(cat "$VERSION_FILE")"
    fi

    if [ ! -f "./$STABLE_ASSET" ] || [ "$local_tag" != "$latest_tag" ]; then
        yellow "Downloading Rock-OS $latest_tag for $PLATFORM $ARCH..."
        VERSIONED_ASSET="rock-os-wiki-$latest_tag-$PLATFORM-$ARCH"
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
                printf '%s' "$latest_tag" > "$VERSION_FILE"
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
    BINARY_SOURCE="stable"
else
    for FILE in $(find . -maxdepth 1 -type f -name "rock-os-wiki-v*-$PLATFORM-$ARCH" | sort -r); do
        BINARY="$FILE"
        BINARY_SOURCE="versioned"
        break
    done
fi

check_git_crypt
check_private
check_go

if [ -n "$BINARY" ] && [ -x "$BINARY" ]; then
    if [ "$BINARY_SOURCE" = "stable" ]; then
        green "Release binary found: $BINARY"
    else
        yellow "Using versioned fallback binary: $BINARY"
    fi

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
    GOCACHE="$PWD/.gocache" go run . --host "$ROCK_OS_HOST"
fi

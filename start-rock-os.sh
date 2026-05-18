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

check_private() {
    private_files="$(git -C .. ls-files 'Website/markdown/Private/**' 2>/dev/null || true)"
    if [ -z "$private_files" ]; then
        green "Private Markdown Folder unlocked."
        return
    fi

    locked_marker="${TMPDIR:-/tmp}/rock-os-private-locked-$$"
    rm -f "$locked_marker"

    printf '%s\n' "$private_files" | while IFS= read -r file; do
        [ -f "../$file" ] || continue

        if dd if="../$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
            printf 'locked' > "$locked_marker"
        fi
    done

    if [ -f "$locked_marker" ]; then
        rm -f "$locked_marker"
        red "Private Markdown Folder locked."
    else
        green "Private Markdown Folder unlocked."
    fi
}

cd "$(dirname "$0")/Website"

REPO="rocketpowerinc/rock-os"
VERSION_FILE=".rock-os-wiki-version"
BINARY=""
BINARY_SOURCE=""

green "[Rock-OS] Launcher online."

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

check_private

if [ -n "$BINARY" ] && [ -x "$BINARY" ]; then
    if [ "$BINARY_SOURCE" = "stable" ]; then
        green "Release binary found: $BINARY"
    else
        yellow "Using versioned fallback binary: $BINARY"
    fi

    green "Starting Rock-OS..."
    "$BINARY" --host local
elif [ -n "$BINARY" ] && [ -f "$BINARY" ]; then
    yellow "Release binary found but is not executable. Making it executable..."
    chmod +x "$BINARY"
    green "Starting Rock-OS..."
    "$BINARY" --host local
else
    yellow "Release binary not found. Starting Rock-OS from Go source..."
    GOCACHE="$PWD/.gocache" go run . --host local
fi

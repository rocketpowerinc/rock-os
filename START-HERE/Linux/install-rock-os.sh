#!/usr/bin/env sh
set -eu

REPO_URL="https://github.com/rocketpowerinc/rock-os.git"
INSTALL_DIR="$HOME/rock-os"
BIN_DIR="$HOME/.local/bin"
ROCK_COMMAND="$BIN_DIR/rock-os"

green() {
    printf '\033[32m%s\033[0m\n' "$1"
}

yellow() {
    printf '\033[33m%s\033[0m\n' "$1"
}

require_git() {
    if ! command -v git >/dev/null 2>&1; then
        printf '%s\n' "Git is required. Install Git, then run this installer again." >&2
        exit 1
    fi
}

ensure_repo() {
    if [ -d "$INSTALL_DIR/.git" ]; then
        green "Rock-OS repo found at $INSTALL_DIR"
        git -C "$INSTALL_DIR" pull --ff-only
        return
    fi

    if [ -e "$INSTALL_DIR" ]; then
        printf '%s\n' "$INSTALL_DIR exists but is not a Git clone. Move it or remove it, then run this installer again." >&2
        exit 1
    fi

    green "Cloning Rock-OS into $INSTALL_DIR"
    git clone "$REPO_URL" "$INSTALL_DIR"
}

add_path_to_profile() {
    profile="$1"
    line='export PATH="$HOME/.local/bin:$PATH"'

    [ -f "$profile" ] || touch "$profile"

    if ! grep -F "$line" "$profile" >/dev/null 2>&1; then
        printf '\n%s\n' "$line" >> "$profile"
        green "Added ~/.local/bin to $profile"
    fi
}

ensure_path() {
    mkdir -p "$BIN_DIR"

    case ":$PATH:" in
        *":$BIN_DIR:"*)
            ;;
        *)
            add_path_to_profile "$HOME/.profile"
            if [ -n "${ZSH_VERSION:-}" ] || [ -f "$HOME/.zshrc" ]; then
                add_path_to_profile "$HOME/.zshrc"
            fi
            if [ -n "${BASH_VERSION:-}" ] || [ -f "$HOME/.bashrc" ]; then
                add_path_to_profile "$HOME/.bashrc"
            fi
            yellow "Open a new terminal before running rock-os if this shell does not see the new PATH yet."
            ;;
    esac
}

write_rock_command() {
    cat > "$ROCK_COMMAND" <<EOF
#!/usr/bin/env sh
exec "$INSTALL_DIR/START-HERE/Linux/start-rock-os.sh" "\$@"
EOF
    chmod +x "$ROCK_COMMAND"
    green "Created terminal command: rock-os"
}

create_linux_desktop_launcher() {
    desktop_dir="$HOME/Desktop"
    launcher="$desktop_dir/Rock-OS.desktop"
    icon="$INSTALL_DIR/Website/assets/icon-512.png"

    [ -d "$desktop_dir" ] || return 0

    cat > "$launcher" <<EOF
[Desktop Entry]
Type=Application
Name=Rock-OS
Comment=Start Rock-OS
Exec=$INSTALL_DIR/START-HERE/Linux/start-rock-os.sh
Icon=$icon
Terminal=true
Categories=Utility;
EOF
    chmod +x "$launcher"
    green "Created Linux desktop launcher: $launcher"
}

create_macos_desktop_launcher() {
    desktop_dir="$HOME/Desktop"
    app_dir="$desktop_dir/Rock-OS.app"
    macos_dir="$app_dir/Contents/MacOS"
    resources_dir="$app_dir/Contents/Resources"
    launcher="$macos_dir/rock-os"
    icon_png="$INSTALL_DIR/Website/assets/icon-512.png"
    iconset="${TMPDIR:-/tmp}/rock-os.iconset"
    icns="$resources_dir/rock-os.icns"

    [ -d "$desktop_dir" ] || return 0

    mkdir -p "$macos_dir" "$resources_dir"

    cat > "$launcher" <<EOF
#!/usr/bin/env sh
osascript <<APPLESCRIPT
tell application "Terminal"
    do script "cd '$INSTALL_DIR/START-HERE/Mac' && ./start-rock-os.sh"
    activate
end tell
APPLESCRIPT
EOF
    chmod +x "$launcher"

    cat > "$app_dir/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>Rock-OS</string>
    <key>CFBundleDisplayName</key>
    <string>Rock-OS</string>
    <key>CFBundleExecutable</key>
    <string>rock-os</string>
    <key>CFBundleIconFile</key>
    <string>rock-os</string>
    <key>CFBundleIdentifier</key>
    <string>com.rocketpowerinc.rock-os</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
</dict>
</plist>
EOF

    if command -v sips >/dev/null 2>&1 && command -v iconutil >/dev/null 2>&1 && [ -f "$icon_png" ]; then
        rm -rf "$iconset"
        mkdir -p "$iconset"
        sips -z 16 16 "$icon_png" --out "$iconset/icon_16x16.png" >/dev/null
        sips -z 32 32 "$icon_png" --out "$iconset/icon_16x16@2x.png" >/dev/null
        sips -z 32 32 "$icon_png" --out "$iconset/icon_32x32.png" >/dev/null
        sips -z 64 64 "$icon_png" --out "$iconset/icon_32x32@2x.png" >/dev/null
        sips -z 128 128 "$icon_png" --out "$iconset/icon_128x128.png" >/dev/null
        sips -z 256 256 "$icon_png" --out "$iconset/icon_128x128@2x.png" >/dev/null
        sips -z 256 256 "$icon_png" --out "$iconset/icon_256x256.png" >/dev/null
        sips -z 512 512 "$icon_png" --out "$iconset/icon_256x256@2x.png" >/dev/null
        sips -z 512 512 "$icon_png" --out "$iconset/icon_512x512.png" >/dev/null
        cp "$icon_png" "$iconset/icon_512x512@2x.png"
        iconutil -c icns "$iconset" -o "$icns" >/dev/null
        rm -rf "$iconset"
    else
        cp "$icon_png" "$resources_dir/rock-os.png" 2>/dev/null || true
        yellow "Could not build a macOS .icns icon. The app launcher was still created."
    fi

    green "Created macOS desktop app: $app_dir"
}

require_git
ensure_repo
ensure_path
write_rock_command

case "$(uname -s)" in
    Darwin)
        create_macos_desktop_launcher
        ;;
    *)
        create_linux_desktop_launcher
        ;;
esac

green ""
green "Rock-OS is installed."
green "Run it from a new terminal with: rock-os"
green "Or use the Rock-OS desktop launcher."
green "Starting Rock-OS now..."

exec sh "$INSTALL_DIR/START-HERE/Linux/start-rock-os.sh"

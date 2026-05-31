#!/usr/bin/env sh
set -eu

# Firefox reads enterprise policies from distribution/policies.json at startup.
# This script merges a small Rock-OS policy into that file instead of editing
# Firefox's profile database directly, which is safer and easier to review..

printf '%s\n' "This script configures Firefox with a small Rock-OS policy."
printf '%s\n' "It will:"
printf '%s\n' "  - Always show the bookmarks toolbar"
printf '%s\n' "  - Remove Firefox's default import-bookmarks toolbar prompt"
printf '%s\n' "  - Add an Xbox bookmark to the toolbar"
printf '%s\n' "  - Install uBlock Origin from Mozilla Add-ons"
printf '%s\n' ""
printf '%s\n' "Rock-OS opens this script in your OS terminal so sudo prompts work normally."
printf '%s\n' "Close Firefox before running this script so policies reload cleanly."
printf '%s\n' ""

if ! command -v firefox >/dev/null 2>&1 && ! command -v firefox-esr >/dev/null 2>&1; then
    printf '%s\n' "Firefox was not found. Install Firefox, then run this script again."
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    printf '%s\n' "python3 is required to safely merge Firefox policies."
    exit 1
fi

POLICY_DIR=""

# Firefox policy location varies by distro/package, so check the common paths
# first and fall back to the standard /usr/lib/firefox location.
for candidate in \
    "/usr/lib/firefox/distribution" \
    "/usr/lib64/firefox/distribution" \
    "/usr/lib/firefox-esr/distribution" \
    "/opt/firefox/distribution"
do
    if [ -d "$candidate" ]; then
        POLICY_DIR="$candidate"
        break
    fi
done

if [ -z "$POLICY_DIR" ]; then
    POLICY_DIR="/usr/lib/firefox/distribution"
fi

TEMP_POLICY="$(mktemp)"
EXISTING_POLICY="$POLICY_DIR/policies.json"

if [ -f "$EXISTING_POLICY" ]; then
    sudo cp "$EXISTING_POLICY" "$TEMP_POLICY.existing"
else
    printf '%s\n' '{}' > "$TEMP_POLICY.existing"
fi

python3 - "$TEMP_POLICY.existing" "$TEMP_POLICY" <<'PY'
import json
import sys

existing_path, output_path = sys.argv[1], sys.argv[2]

# Preserve any existing policy settings, then add only the Firefox settings this
# script owns: toolbar visibility, default bookmark/import cleanup, one bookmark,
# and uBlock Origin installation.
try:
    with open(existing_path, "r", encoding="utf-8") as source:
        data = json.load(source)
except json.JSONDecodeError:
    data = {}

policies = data.setdefault("policies", {})

policies["DisplayBookmarksToolbar"] = True
policies["NoDefaultBookmarks"] = True
policies["DisableProfileImport"] = True

bookmarks = policies.setdefault("Bookmarks", [])
xbox_bookmark = {
    "Title": "Xbox",
    "URL": "https://www.xbox.com/",
    "Placement": "toolbar"
}

if not any(
    item.get("URL") == xbox_bookmark["URL"]
    for item in bookmarks
    if isinstance(item, dict)
):
    bookmarks.append(xbox_bookmark)

extension_settings = policies.setdefault("ExtensionSettings", {})
extension_settings["uBlock0@raymondhill.net"] = {
    "installation_mode": "force_installed",
    "install_url": "https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi"
}

with open(output_path, "w", encoding="utf-8") as target:
    json.dump(data, target, indent=2)
    target.write("\n")
PY

sudo mkdir -p "$POLICY_DIR"

if [ -f "$EXISTING_POLICY" ]; then
    backup="$EXISTING_POLICY.rock-os-backup.$(date +%Y%m%d-%H%M%S)"
    sudo cp "$EXISTING_POLICY" "$backup"
    printf '%s\n' "Backed up existing policy to $backup"
fi

sudo install -m 0644 "$TEMP_POLICY" "$EXISTING_POLICY"

rm -f "$TEMP_POLICY"
sudo rm -f "$TEMP_POLICY.existing"

# NoDefaultBookmarks stops Firefox from adding the prompt in new profiles, but
# existing profiles may already have it saved in places.sqlite. Remove only that
# default import bookmark from each local profile, backing up first.
python3 - <<'PY'
import glob
import os
import shutil
import sqlite3
import time

profile_root = os.path.expanduser("~/.mozilla/firefox")
databases = glob.glob(os.path.join(profile_root, "*.default*", "places.sqlite"))
titles = (
    "Import bookmarks from another browser",
    "Import Bookmarks from Another Browser",
)

if not databases:
    print("No Firefox profile bookmark databases found to clean.")
else:
    removed_total = 0
    for database in databases:
        backup = f"{database}.rock-os-backup.{time.strftime('%Y%m%d-%H%M%S')}"
        try:
            shutil.copy2(database, backup)
            connection = sqlite3.connect(database)
            try:
                cursor = connection.cursor()
                cursor.execute(
                    """
                    DELETE FROM moz_bookmarks
                    WHERE type = 1
                      AND (
                        title IN (?, ?)
                        OR lower(title) LIKE 'import bookmarks from another browser%'
                      )
                    """,
                    titles,
                )
                removed = cursor.rowcount
                connection.commit()
            finally:
                connection.close()

            removed_total += max(removed, 0)
            print(f"Cleaned {removed} import bookmark(s) from {database}")
            print(f"Bookmark database backup saved to {backup}")
        except sqlite3.OperationalError as error:
            print(f"Could not clean {database}: {error}")
            print("Close Firefox completely, then run this script again.")

    if removed_total == 0:
        print("No existing import-bookmarks toolbar prompt was found.")
PY

printf '%s\n' ""
printf '%s\n' "Firefox policy installed:"
printf '%s\n' "$EXISTING_POLICY"
printf '%s\n' ""
printf '%s\n' "Restart Firefox, then open about:policies to verify it loaded."

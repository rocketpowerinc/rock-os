#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

rocket_markdown_locked() {
    git ls-files -- 'Website/menu/rocket' | while IFS= read -r file; do
        [ -f "$file" ] || continue
        if dd if="$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
            exit 2
        fi
    done
    [ "$?" -eq 2 ]
}

verify_rocket_unlocked() {
    if ! rocket_markdown_locked; then
        echo "Rocket markdown verified unlocked."
        return 0
    fi

    echo "Rocket files still look encrypted. Refreshing clean Rocket files..."
    if [ -n "$(git status --porcelain -- 'Website/menu/rocket')" ]; then
        echo "Rocket markdown has local changes, so this script will not restore it automatically."
        echo "Back up or clear those changes first, then run:"
        echo "git restore --source=HEAD --worktree -- Website/menu/rocket"
        return 1
    fi

    if ! git restore --source=HEAD --worktree -- 'Website/menu/rocket' 2>/dev/null; then
        git checkout -- 'Website/menu/rocket'
    fi

    if rocket_markdown_locked; then
        echo "Rocket markdown still looks encrypted after refresh."
        return 1
    fi

    echo "Rocket markdown verified unlocked."
    return 0
}

if ! command -v git-crypt >/dev/null 2>&1; then
    echo "git-crypt was not found."
    echo "Install git-crypt, then run this script again."
    echo "Repo root: $REPO_ROOT"
    exit 1
fi

set -- ./*.key

if [ "$1" = "./*.key" ]; then
    echo "No .key file was found in the repo root:"
    echo "$REPO_ROOT"
    echo "Copy your exported git-crypt key to the repo root folder, then run this script again."
    exit 1
fi

if [ "$#" -gt 1 ]; then
    echo "More than one .key file was found in the repo root:"
    echo "$REPO_ROOT"
    echo "Keep only the git-crypt key in that folder, then run this script again."
    exit 1
fi

echo "Unlocking repository with $1..."
key_name="$(basename "$1")"
temp_key="${TMPDIR:-/tmp}/rock-os-git-crypt-$$.key"
cp "$1" "$temp_key"
rm "$1"
set +e
git-crypt unlock "$temp_key"
unlock_result=$?
set -e
if ! cp "$temp_key" "./$key_name"; then
    echo "Failed to copy the key back to the repo root."
    echo "Your key is still at $temp_key."
    exit 1
fi
rm -f "$temp_key"
if [ "$unlock_result" -ne 0 ]; then
    echo "Failed to unlock the repository."
    exit "$unlock_result"
fi

verify_rocket_unlocked

echo "Repository unlocked."
echo "Key restored to $REPO_ROOT/$key_name."

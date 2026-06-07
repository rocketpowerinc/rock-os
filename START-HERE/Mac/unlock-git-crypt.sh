#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

encrypted_content_locked() {
    git ls-files -- 'Website/ENCRYPTED' | while IFS= read -r file; do
        [ -f "$file" ] || continue
        if dd if="$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
            exit 2
        fi
    done
    [ "$?" -eq 2 ]
}

verify_encrypted_content_unlocked() {
    if ! encrypted_content_locked; then
        echo "Encrypted content verified unlocked."
        return 0
    fi

    echo "Encrypted content files still look encrypted. Refreshing clean Encrypted content files..."
    if [ -n "$(git status --porcelain -- 'Website/ENCRYPTED')" ]; then
        echo "Encrypted content has local changes, so this script will not restore it automatically."
        echo "Back up or clear those changes first, then run:"
        echo "git restore --source=HEAD --worktree -- Website/ENCRYPTED"
        return 1
    fi

    git ls-files -- 'Website/ENCRYPTED' | while IFS= read -r file; do
        [ -f "$file" ] && rm -f -- "$file"
    done

    if ! git restore --source=HEAD --worktree -- 'Website/ENCRYPTED' 2>/dev/null; then
        git checkout -- 'Website/ENCRYPTED'
    fi

    if encrypted_content_locked; then
        echo "Encrypted content still looks encrypted after refresh."
        return 1
    fi

    echo "Encrypted content verified unlocked."
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
    echo "No git-crypt .key file was found in the repo root:"
    echo "$REPO_ROOT"
    echo "Copy your exported git-crypt key to the repo root folder, then run this script again."
    exit 1
fi

git_crypt_keys=
for key_file do
    git_crypt_keys="${git_crypt_keys}${git_crypt_keys:+
}$key_file"
done

if [ -z "$git_crypt_keys" ]; then
    echo "No git-crypt .key file was found in the repo root:"
    echo "$REPO_ROOT"
    echo "Copy your exported git-crypt key to the repo root folder, then run this script again."
    exit 1
fi

key_count=$(printf '%s\n' "$git_crypt_keys" | wc -l | tr -d ' ')
if [ "$key_count" -gt 1 ]; then
    echo "More than one git-crypt .key file was found in the repo root:"
    echo "$REPO_ROOT"
    echo "Keep only one git-crypt key in that folder, then run this script again."
    exit 1
fi

key_file=$git_crypt_keys
echo "Unlocking repository with $key_file..."
key_name="$(basename "$key_file")"
temp_key="${TMPDIR:-/tmp}/rock-os-git-crypt-$$.key"
cp "$key_file" "$temp_key"
rm "$key_file"
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
    echo "If you see 'unable to write key file', check permissions on .git/git-crypt/keys."
    exit "$unlock_result"
fi

verify_encrypted_content_unlocked

echo "Repository unlocked."
echo "Key restored to $REPO_ROOT/$key_name."

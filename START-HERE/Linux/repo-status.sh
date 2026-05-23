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

section() {
    printf '\n'
    green "-- $1 --"
}

ok() {
    green "[OK] $1"
}

info() {
    green "[INFO] $1"
}

warn() {
    yellow "[WARN] $1"
}

bad() {
    red "[BAD] $1"
}

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

printf '\n'
green "== Rock-OS Repo Status =="

if [ ! -e ".git" ]; then
    bad "This folder is not a cloned Git repository."
    warn "Use: git clone https://github.com/rocketpowerinc/rock-os.git"
    exit 1
fi

section "Git"
if command -v git >/dev/null 2>&1; then
    ok "$(git --version)"
else
    bad "Git is not installed or not on PATH."
    exit 1
fi

branch="$(git branch --show-current 2>/dev/null || true)"
[ -n "$branch" ] || branch="(detached HEAD)"
info "Branch: $branch"
info "Commit: $(git rev-parse --short HEAD 2>/dev/null || printf 'unknown')"
info "Total commits: $(git rev-list --count HEAD 2>/dev/null || printf 'unknown')"

status_short="$(git status --short 2>/dev/null || true)"
if [ -n "$status_short" ]; then
    warn "Working tree has changes:"
    printf '%s\n' "$status_short" | sed 's/^/  /'
else
    ok "Working tree clean."
fi

section "Website"
if find Website -maxdepth 1 -type f -name 'rock-os-wiki-*' | grep -q .; then
    ok "Release binary present. Site can run without Go installed."
else
    warn "No release binary found in Website folder."
fi

[ -f "cmd/rock-os-wiki/main.go" ] && ok "Go server source present for source fallback." || bad "cmd/rock-os-wiki/main.go missing."

if command -v go >/dev/null 2>&1; then
    ok "$(go version)"
else
    warn "Go is not installed or not on PATH. Not needed if using release binary."
fi

section "Port 8000"
pids=""
if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -tiTCP:8000 -sTCP:LISTEN 2>/dev/null || true)"
elif command -v ss >/dev/null 2>&1; then
    pids="$(ss -ltnp 'sport = :8000' 2>/dev/null | awk -F'pid=' 'NF > 1 { split($2, parts, ","); print parts[1] }' | sort -u)"
elif command -v netstat >/dev/null 2>&1; then
    pids="$(netstat -ltnp 2>/dev/null | awk '$4 ~ /:8000$/ { split($7, parts, "/"); print parts[1] }' | sort -u)"
fi

if [ -n "$pids" ]; then
    printf '%s\n' "$pids" | while IFS= read -r pid; do
        [ -n "$pid" ] || continue
        ok "Port 8000 is listening on PID $pid."
    done
else
    info "Port 8000 is not currently listening."
fi

section "git-crypt"
if command -v git-crypt >/dev/null 2>&1; then
    ok "git-crypt installed."
    if git-crypt status >/dev/null 2>&1; then
        ok "git-crypt status is available."
    else
        warn "git-crypt status could not be read."
    fi
else
    bad "git-crypt is not installed."
fi

private_files="$(git ls-files -- 'Website/tabs/rocket' 2>/dev/null || true)"
private_found=""
private_locked=""

printf '%s\n' "$private_files" | while IFS= read -r file; do
    [ -n "$file" ] || continue
    [ -f "$file" ] || continue
    printf 'found\n' > "${TMPDIR:-/tmp}/rock-os-private-found-$$"
    if dd if="$file" bs=16 count=1 2>/dev/null | grep -a -q 'GITCRYPT'; then
        printf 'locked\n' > "${TMPDIR:-/tmp}/rock-os-private-locked-$$"
    fi
done

if [ -f "${TMPDIR:-/tmp}/rock-os-private-found-$$" ]; then
    private_found="true"
    rm -f "${TMPDIR:-/tmp}/rock-os-private-found-$$"
fi

if [ -f "${TMPDIR:-/tmp}/rock-os-private-locked-$$" ]; then
    private_locked="true"
    rm -f "${TMPDIR:-/tmp}/rock-os-private-locked-$$"
fi

if [ -z "$private_found" ]; then
    info "No tracked Rocket wiki files found."
elif [ -n "$private_locked" ]; then
    bad "Rocket Wiki Folder Locked."
else
    ok "Rocket Wiki Folder Unlocked."
fi

key_count="$(find . -maxdepth 1 -type f -name '*.key' | wc -l | tr -d ' ')"
if [ "$key_count" -gt 0 ]; then
    warn ".key file present in repo root. Keep it private and never commit it."
else
    ok "No .key files found in repo root."
fi

section "Full git-crypt status"
if command -v git-crypt >/dev/null 2>&1; then
    git-crypt status || warn "git-crypt status exited with an error."
else
    bad "git-crypt is not installed."
fi

section "Done"
ok "Repo status check complete."

#!/usr/bin/env sh
set -eu

# Restores executable permissions on all shell scripts after cloning or
# extracting the repo on Linux or macOS.

cd "$(dirname "$0")"

find . -type f -name '*.sh' -exec chmod +x {} +

printf '%s\n' "Executable permissions applied to all .sh files."

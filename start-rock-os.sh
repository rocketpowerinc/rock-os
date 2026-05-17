#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/Website"
go run . --host local

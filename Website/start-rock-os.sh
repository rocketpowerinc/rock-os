#!/usr/bin/env sh
cd "$(dirname "$0")" || exit 1
go run . --host local

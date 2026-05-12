#!/usr/bin/env bash
set -euo pipefail

prefix="${PREFIX:-$HOME/.local}"
bindir="$prefix/bin"

mkdir -p "$bindir"
go build -o "$bindir/weazlwrite" ./cmd/weazlwrite

printf 'installed %s\n' "$bindir/weazlwrite"

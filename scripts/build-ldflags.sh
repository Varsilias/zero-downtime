#!/usr/bin/env bash
set -euo pipefail

export OLLAMA_WAIT="false"

VERSION="${VERSION:-$(git describe --tags --always --dirty=-dirty 2>/dev/null || echo 1.0.0)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
BUILT_AT="${BUILT_AT:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

# <-- MUST match go.mod module path
PKG="github.com/varsilias/zero-downtime/internal/buildinfo"

# <-- MUST match .air.toml [build].bin and full_bin
OUT="${OUT:-./tmp/main}"

LD="-X ${PKG}.Version=${VERSION} -X ${PKG}.Commit=${COMMIT} -X ${PKG}.BuiltAt=${BUILT_AT}"

mkdir -p "$(dirname "$OUT")"

# entry file is root `main.go`,
go build -ldflags "$LD" -o "$OUT" .

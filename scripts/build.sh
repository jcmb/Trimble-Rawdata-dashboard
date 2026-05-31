#!/usr/bin/env bash
# Cross-compile trimble-rawdata-dashboard for common platforms.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PKG="./cmd/trimble-rawdata-dashboard"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X github.com/gkirk/trimble-rawdata-dashboard/internal/version.Build=${VERSION}"

mkdir -p "$OUT_DIR"

build() {
  local goos="$1"
  local goarch="$2"
  local out="$3"
  echo "→ ${goos}/${goarch} → ${out}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/$out" "$PKG"
}

build windows amd64 trimble-rawdata-dashboard-windows-amd64.exe
build darwin  arm64 trimble-rawdata-dashboard-darwin-arm64
build linux   amd64 trimble-rawdata-dashboard-linux-amd64
echo "→ linux/arm (GOARM=7) → trimble-rawdata-dashboard-linux-arm32"
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT_DIR/trimble-rawdata-dashboard-linux-arm32" "$PKG"

echo
echo "Built into $OUT_DIR:"
ls -lh "$OUT_DIR"/trimble-rawdata-dashboard-*

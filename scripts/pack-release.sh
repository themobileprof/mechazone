#!/usr/bin/env bash
# Build a shop-folder archive: compiled Go ledger + shop screen, no Go/Node required on the laptop.
# Usage: ./scripts/pack-release.sh [linux|windows] [amd64|arm64]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

OS="${1:-linux}"
ARCH="${2:-amd64}"
case "$OS" in
  linux|windows) ;;
  *) printf 'OS must be linux or windows\n' >&2; exit 1 ;;
esac
case "$ARCH" in
  amd64|arm64) ;;
  *) printf 'ARCH must be amd64 or arm64\n' >&2; exit 1 ;;
esac

host_arch="$(uname -m)"
native_arch=amd64
case "$host_arch" in
  x86_64) native_arch=amd64 ;;
  aarch64|arm64) native_arch=arm64 ;;
esac

OUT="$ROOT/var/release"
STAGE="$OUT/stage-$OS-$ARCH"
NAME="mechazone-$OS-$ARCH"
PREFIX="$STAGE/mechazone"

rm -rf "$STAGE"
mkdir -p "$PREFIX/bin" "$PREFIX/passthru" "$PREFIX/var" "$PREFIX/data/imported-reports" \
  "$PREFIX/cloud-backend/seeds" "$PREFIX/scripts" "$PREFIX/deploy" "$PREFIX/docs" \
  "$PREFIX/diagnostic-worker"

say() { printf '==> %s\n' "$*"; }

say "Shop screen"
(
  cd "$ROOT/client"
  if [[ ! -d node_modules ]]; then
    if [[ -f package-lock.json ]]; then
      npm ci --no-fund --no-audit
    else
      npm install --no-fund --no-audit
    fi
  fi
  npm run build
)
mkdir -p "$PREFIX/client"
cp -a "$ROOT/client/dist" "$PREFIX/client/dist"

say "Ledger ($OS/$ARCH)"
ext=""
[[ "$OS" == windows ]] && ext=".exe"
(
  cd "$ROOT/cloud-backend"
  CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" GOTOOLCHAIN=local go build -trimpath -ldflags '-s -w' \
    -o "$PREFIX/bin/mechazone-server$ext" ./cmd/server
)

say "Worker source"
tar -C "$ROOT/diagnostic-worker" --exclude .venv --exclude __pycache__ --exclude '*.pyc' --exclude wheels --exclude .pytest_cache \
  -cf - . | tar -C "$PREFIX/diagnostic-worker" -xf -

if [[ "$OS" == linux ]]; then
  say "Python wheels (so the shop laptop can pip-install offline)"
  mkdir -p "$PREFIX/diagnostic-worker/wheels"
  python3 -m pip download -q -d "$PREFIX/diagnostic-worker/wheels" -r "$ROOT/diagnostic-worker/requirements.txt"
fi

if [[ "$OS" == linux && "$ARCH" == "$native_arch" ]]; then
  say "OpenPort Linux library"
  if [[ ! -d "$ROOT/third_party/j2534" ]]; then
    git clone --depth 1 https://github.com/NikolaKozina/j2534.git "$ROOT/third_party/j2534"
  fi
  if command -v pkg-config >/dev/null && pkg-config --exists libusb-1.0 2>/dev/null; then
    make -C "$ROOT/third_party/j2534/j2534"
    cp "$ROOT/third_party/j2534/j2534/j2534.so" "$PREFIX/passthru/j2534.so"
    if [[ -f "$ROOT/third_party/j2534/LICENSE" ]]; then
      cp "$ROOT/third_party/j2534/LICENSE" "$PREFIX/passthru/j2534-LICENSE"
    fi
  else
    printf '    skip j2534.so (libusb-1.0-dev not installed)\n'
  fi
fi
: > "$PREFIX/passthru/.gitkeep"
: > "$PREFIX/data/imported-reports/.gitkeep"

cp "$ROOT/install.sh" "$ROOT/install.ps1" "$ROOT/.env.example" "$ROOT/README.md" "$PREFIX/"
cp "$ROOT/docs/install.md" "$ROOT/docs/integrations.md" "$PREFIX/docs/"
cp "$ROOT/deploy/99-openport.rules" "$PREFIX/deploy/"
cp "$ROOT/scripts/start-mechazone.sh" "$ROOT/scripts/stop-mechazone.sh" "$PREFIX/scripts/"
cp "$ROOT/scripts/start-mechazone.ps1" "$ROOT/scripts/stop-mechazone.ps1" "$PREFIX/scripts/"
cp "$ROOT/cloud-backend/seeds/"*.csv "$PREFIX/cloud-backend/seeds/" 2>/dev/null || true
chmod +x "$PREFIX/install.sh" "$PREFIX/scripts/"*.sh

mkdir -p "$OUT"
(
  cd "$STAGE"
  if [[ "$OS" == windows ]]; then
    zip -qr "$OUT/$NAME.zip" mechazone
    say "Wrote $OUT/$NAME.zip"
  else
    tar -czf "$OUT/$NAME.tar.gz" mechazone
    say "Wrote $OUT/$NAME.tar.gz"
  fi
)

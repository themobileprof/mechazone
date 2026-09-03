#!/usr/bin/env bash
# Mechazone shop installer — run once on the laptop that will sit in the bay.
#
# Ubuntu / Debian:
#   1. Open this folder
#   2. Right-click empty space → Open in Terminal
#   3. Type:  ./install.sh
#   4. Press Enter. Type your computer password if asked.
#
# A GitHub Release zip already contains the ledger binary and shop screen.
# You do not need Go or Node in that case — only Python, PostgreSQL, and USB permission.
#
# After it finishes, double-click Mechazone on the Desktop.
# Full guide: docs/install.md
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
TOOLS="$ROOT/.tools"
mkdir -p "$TOOLS" "$ROOT/bin" "$ROOT/var"

say() { printf '\n==> %s\n' "$*"; }
ok() { printf '    %s\n' "$*"; }
die() { printf '\nERROR: %s\n' "$*" >&2; exit 1; }

export PATH="$TOOLS/go/bin:$TOOLS/node/bin:$PATH"

prebuilt=0
if [[ -x "$ROOT/bin/mechazone-server" && -f "$ROOT/client/dist/index.html" ]]; then
  prebuilt=1
fi

say "Mechazone installer"
if ((prebuilt)); then
  ok "This folder already has a compiled ledger. No Go or Node on this laptop."
else
  ok "Source tree: this will download Go/Node if needed and compile once."
fi
ok "You only do this once."

arch="$(uname -m)"
case "$arch" in
  x86_64) go_arch=amd64; node_arch=x64 ;;
  aarch64|arm64) go_arch=arm64; node_arch=arm64 ;;
  *) die "This installer supports 64-bit Intel/AMD and ARM laptops. This machine reports: $arch" ;;
esac

need_apt=()
command -v python3 >/dev/null || need_apt+=(python3 python3-venv python3-pip)
command -v psql >/dev/null || need_apt+=(postgresql postgresql-client)
command -v curl >/dev/null || need_apt+=(curl ca-certificates)
if command -v dpkg >/dev/null; then
  dpkg -s libusb-1.0-0 >/dev/null 2>&1 || need_apt+=(libusb-1.0-0)
fi
if ! ((prebuilt)); then
  command -v git >/dev/null || need_apt+=(git)
  command -v make >/dev/null || need_apt+=(build-essential)
  if command -v dpkg >/dev/null; then
    dpkg -s libusb-1.0-0-dev >/dev/null 2>&1 || need_apt+=(libusb-1.0-0-dev pkg-config)
  fi
fi
if [[ ! -f "$ROOT/passthru/j2534.so" && ! -f "$ROOT/third_party/j2534/j2534/j2534.so" ]]; then
  command -v make >/dev/null || need_apt+=(build-essential)
  command -v git >/dev/null || need_apt+=(git)
  if command -v dpkg >/dev/null; then
    dpkg -s libusb-1.0-0-dev >/dev/null 2>&1 || need_apt+=(libusb-1.0-0-dev pkg-config)
  fi
fi

if ((${#need_apt[@]})); then
  say "Installing laptop pieces (you may be asked for your password)"
  ok "Missing: ${need_apt[*]}"
  command -v sudo >/dev/null || die "Ask whoever set up this laptop to install: ${need_apt[*]}"
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "${need_apt[@]}"
fi

command -v python3 >/dev/null || die "Python 3 is required."
command -v psql >/dev/null || die "PostgreSQL is required. Install it, then run this file again."
command -v curl >/dev/null || die "curl is required."

go_new_enough() {
  command -v go >/dev/null || return 1
  local ver major minor
  ver="$(go version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | tr -d 'go')"
  [[ -n "$ver" ]] || return 1
  major="${ver%%.*}"
  minor="${ver#*.}"
  (( major > 1 || (major == 1 && minor >= 24) ))
}

node_new_enough() {
  command -v node >/dev/null || return 1
  local major
  major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || true)"
  [[ -n "$major" ]] && (( major >= 20 ))
}

if ! ((prebuilt)); then
  if ! go_new_enough; then
    say "Downloading Go (the ledger program needs it)"
    go_ver="1.24.6"
    tarball="go${go_ver}.linux-${go_arch}.tar.gz"
    curl -fsSL "https://go.dev/dl/${tarball}" -o "$TOOLS/$tarball"
    rm -rf "$TOOLS/go"
    tar -C "$TOOLS" -xzf "$TOOLS/$tarball"
    rm -f "$TOOLS/$tarball"
    export PATH="$TOOLS/go/bin:$PATH"
    ok "Go $(go version | awk '{print $3}')"
  fi

  if ! node_new_enough; then
    say "Downloading Node (the shop screen needs it)"
    node_ver="22.18.0"
    tarball="node-v${node_ver}-linux-${node_arch}.tar.xz"
    curl -fsSL "https://nodejs.org/dist/v${node_ver}/${tarball}" -o "$TOOLS/$tarball"
    rm -rf "$TOOLS/node"
    tar -C "$TOOLS" -xJf "$TOOLS/$tarball"
    mv "$TOOLS/node-v${node_ver}-linux-${node_arch}" "$TOOLS/node"
    rm -f "$TOOLS/$tarball"
    export PATH="$TOOLS/node/bin:$PATH"
    ok "Node $(node -v)"
  fi

  command -v go >/dev/null || die "Go did not install. Check the internet connection and try again."
  command -v node >/dev/null || die "Node did not install. Check the internet connection and try again."
  command -v npm >/dev/null || die "npm is missing (it should come with Node)."
fi

if command -v pg_isready >/dev/null && ! pg_isready -q 2>/dev/null; then
  say "Starting the database"
  sudo service postgresql start || true
fi

say "Creating the local database (if needed)"
if ! psql -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='mechazone'" 2>/dev/null | grep -q 1; then
  createdb mechazone || die "Could not create the mechazone database. See docs/install.md (Troubleshooting)."
fi
ok "Database: mechazone"

say "Writing settings file"
if [[ ! -f .env ]]; then
  cp .env.example .env
fi
if grep -q '^UI_DIR=' .env; then
  sed -i "s|^UI_DIR=.*|UI_DIR=$ROOT/client/dist|" .env
else
  printf '\nUI_DIR=%s/client/dist\n' "$ROOT" >> .env
fi
ok "Settings: $ROOT/.env"

say "Installing the car-talking worker"
python3 -m venv "$ROOT/diagnostic-worker/.venv"
"$ROOT/diagnostic-worker/.venv/bin/pip" install -q --upgrade pip
if [[ -d "$ROOT/diagnostic-worker/wheels" ]] && compgen -G "$ROOT/diagnostic-worker/wheels/"* >/dev/null; then
  "$ROOT/diagnostic-worker/.venv/bin/pip" install -q --no-index --find-links "$ROOT/diagnostic-worker/wheels" -r "$ROOT/diagnostic-worker/requirements.txt"
else
  "$ROOT/diagnostic-worker/.venv/bin/pip" install -q -r "$ROOT/diagnostic-worker/requirements.txt"
fi
ok "Worker ready"

if ! ((prebuilt)); then
  say "Building the shop screen (a few minutes the first time)"
  (
    cd "$ROOT/client"
    if [[ -f package-lock.json ]]; then
      npm ci --no-fund --no-audit
    else
      npm install --no-fund --no-audit
    fi
    npm run build
  )
  ok "Shop screen ready"

  say "Building the ledger program"
  (
    cd "$ROOT/cloud-backend"
    GOTOOLCHAIN=local go build -ldflags '-s -w' -o "$ROOT/bin/mechazone-server" ./cmd/server
  )
  ok "Ledger ready"
else
  ok "Ledger binary: $ROOT/bin/mechazone-server"
  ok "Shop screen: $ROOT/client/dist"
fi

say "OpenPort USB permission (optional)"
if [[ -f "$ROOT/deploy/99-openport.rules" ]] && command -v sudo >/dev/null; then
  sudo cp "$ROOT/deploy/99-openport.rules" /etc/udev/rules.d/99-openport.rules || true
  sudo udevadm control --reload || true
  sudo usermod -aG dialout "$USER" || true
  ok "USB rule installed. Log out and back in before using the OpenPort cable."
fi

set_j2534() {
  local so="$1"
  if grep -q '^J2534_LIB=' .env; then
    sed -i "s|^J2534_LIB=.*|J2534_LIB=$so|" .env
  else
    printf 'J2534_LIB=%s\n' "$so" >> .env
  fi
  ok "OpenPort library: $so"
}

if [[ -f "$ROOT/passthru/j2534.so" ]]; then
  set_j2534 "$ROOT/passthru/j2534.so"
elif [[ -f "$ROOT/third_party/j2534/j2534/j2534.so" ]]; then
  set_j2534 "$ROOT/third_party/j2534/j2534/j2534.so"
else
  if [[ ! -d "$ROOT/third_party/j2534" ]] && command -v git >/dev/null; then
    git clone --depth 1 https://github.com/NikolaKozina/j2534.git "$ROOT/third_party/j2534" || true
  fi
  if [[ -d "$ROOT/third_party/j2534/j2534" ]] && command -v pkg-config >/dev/null && pkg-config --exists libusb-1.0 2>/dev/null; then
    make -C "$ROOT/third_party/j2534/j2534" || true
    if [[ -f "$ROOT/third_party/j2534/j2534/j2534.so" ]]; then
      mkdir -p "$ROOT/passthru"
      cp "$ROOT/third_party/j2534/j2534/j2534.so" "$ROOT/passthru/j2534.so"
      set_j2534 "$ROOT/passthru/j2534.so"
    fi
  fi
fi

chmod +x "$ROOT/scripts/start-mechazone.sh" "$ROOT/scripts/stop-mechazone.sh" "$ROOT/install.sh"

say "Creating a Start icon"
APP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
mkdir -p "$APP_DIR"
cat > "$APP_DIR/mechazone.desktop" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Mechazone
Comment=Shop diagnostic bay
Exec=$ROOT/scripts/start-mechazone.sh
Path=$ROOT
Terminal=false
Categories=Utility;
StartupNotify=true
EOF
chmod +x "$APP_DIR/mechazone.desktop"
if [[ -d "$HOME/Desktop" ]]; then
  cp "$APP_DIR/mechazone.desktop" "$HOME/Desktop/Mechazone.desktop" 2>/dev/null || true
  chmod +x "$HOME/Desktop/Mechazone.desktop" 2>/dev/null || true
  ok "Desktop icon: Mechazone"
fi
ok "App menu: Mechazone"

say "Done"
ok "Start: double-click Mechazone on the Desktop"
ok "Or run:  $ROOT/scripts/start-mechazone.sh"
ok "First login:     admin@mechazone.local"
ok "First password:  change-me-now"
ok "Change that password after the first login. Guide: docs/install.md"
printf '\n'

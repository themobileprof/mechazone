#!/usr/bin/env bash
# Starts the bay app and opens the shop screen in the browser.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p "$ROOT/var"
export PATH="$ROOT/.tools/go/bin:$ROOT/.tools/node/bin:$PATH"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "$ROOT/.env"
  set +a
fi

export UI_DIR="${UI_DIR:-$ROOT/client/dist}"
export DATABASE_URL="${DATABASE_URL:-postgres:///mechazone?sslmode=disable}"
export HTTP_ADDR="${HTTP_ADDR:-:8080}"

SERVER_BIN="$ROOT/bin/mechazone-server"
WORKER_PY="$ROOT/diagnostic-worker/.venv/bin/python"

if [[ ! -x "$SERVER_BIN" ]]; then
  printf 'Mechazone is not installed yet.\nRun ./install.sh first. See docs/install.md.\n' >&2
  exit 1
fi

if command -v pg_isready >/dev/null && ! pg_isready -q 2>/dev/null; then
  sudo service postgresql start >/dev/null 2>&1 || true
fi

if [[ -f "$ROOT/var/server.pid" ]] && kill -0 "$(cat "$ROOT/var/server.pid")" 2>/dev/null; then
  :
else
  nohup "$SERVER_BIN" >"$ROOT/var/server.log" 2>&1 &
  echo $! >"$ROOT/var/server.pid"
fi

if [[ -x "$WORKER_PY" ]]; then
  if [[ -f "$ROOT/var/worker.pid" ]] && kill -0 "$(cat "$ROOT/var/worker.pid")" 2>/dev/null; then
    :
  else
    (
      cd "$ROOT/diagnostic-worker"
      nohup "$WORKER_PY" -m mechazone_worker >"$ROOT/var/worker.log" 2>&1 &
      echo $! >"$ROOT/var/worker.pid"
    )
  fi
fi

ready=0
for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  if curl -sf http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 0.4
done

if ((ready)); then
  if command -v xdg-open >/dev/null; then
    xdg-open http://127.0.0.1:8080 >/dev/null 2>&1 || true
  elif command -v gio >/dev/null; then
    gio open http://127.0.0.1:8080 >/dev/null 2>&1 || true
  fi
else
  printf 'Mechazone did not start. Open var/server.log or see docs/install.md (Troubleshooting).\n' >&2
  if command -v notify-send >/dev/null; then
    notify-send Mechazone "Did not start. See docs/install.md" || true
  fi
  exit 1
fi

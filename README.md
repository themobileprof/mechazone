# Mechazone

Shop-floor diagnostic ledger. Phase 1: OpenPort 2.0 Rev E (or mock ECU) → this shop's jobs on the VIN → session log → closeout.

## Shop install (non-technical)

On the bay laptop, run the installer **once**, then double-click **Mechazone**.

- Linux: `./install.sh`
- Windows: right-click `install.ps1` → Run with PowerShell

Full picture-by-picture guide: **[docs/install.md](docs/install.md)**

The public page is a conversion landing: shops request registration. There is no self-signup. Super admin (`admin@mechazone.local` / `change-me-now`) issues logins from those tickets via **Issued a login?**

## Developer loop

System Postgres on this laptop (`DATABASE_URL=postgres:///mechazone?sslmode=disable` in `.env`):

```bash
createdb mechazone   # once
make backend         # ledger API on :8080 (loads repo-root .env)
make worker          # other terminal — mock ECU / OpenPort websocket
make client          # other terminal — Vite on :5173, proxies /api to :8080
```

`make backend` is `go run ./cmd/server` from `cloud-backend`. All three need to be running: the bay at http://127.0.0.1:5173 talks to the ledger through the Vite proxy.

After `./install.sh`, daily use is `make start` (or the Desktop icon) → http://127.0.0.1:8080 (built UI served by the ledger). `make stop` ends that pair.

Third-party connect guide: `docs/integrations.md` (OpenPort/J2534, vPIC, CarAPI, Vincario, DTC seed, pgvector).

**Refresh kits** autodetects the OpenPort USB IDs (and lists ELM as detect-only). Mock ECU still exercises the Avensis 3ZR-FAE / Valvematic path. Other VINs get an ISO 15765-4 or Toyota probe — see `docs/coverage.md`.

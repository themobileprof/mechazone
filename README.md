# Mechazone

Shop-floor diagnostic ledger. Phase 1: OpenPort 2.0 Rev E (or mock ECU) → VIN history → session log → closeout.

## Shop install (non-technical)

On the bay laptop, run the installer **once**, then double-click **Mechazone**.

- Linux: `./install.sh`
- Windows: right-click `install.ps1` → Run with PowerShell

Full picture-by-picture guide: **[docs/install.md](docs/install.md)**

First login is the seeded super admin (`admin@mechazone.local` / `change-me-now`). That account adds shops and issues technician/freelancer logins. Technicians then sign in to use the bay. There is no self-signup.

## Developer loop

```bash
createdb mechazone   # uses the Postgres already on this laptop
cd cloud-backend && go run ./cmd/server
# other terminals:
make worker
make client
```

`make up` starts an optional Docker Postgres on **5433** if you do not want the system instance.

Bay UI in dev: http://127.0.0.1:5173  
After `./install.sh`, daily use is http://127.0.0.1:8080 (built UI served by the ledger).

Third-party connect guide: `docs/integrations.md` (OpenPort/J2534, vPIC, CarAPI, Vincario, DTC seed, pgvector).

Connect **Mock ECU** to exercise the Avensis 3ZR-FAE / Valvematic path without a car. Switch to **OpenPort 2.0 Rev E** when `libopenport.so` is on the laptop (`J2534_LIB` if it is not on the default path).

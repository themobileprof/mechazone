# Mechazone

Shop-floor diagnostic ledger. Phase 1: OpenPort 2.0 Rev E (or mock ECU) → VIN history → session log → closeout.

```bash
createdb mechazone   # uses the Postgres already on this laptop
cd cloud-backend && go run ./cmd/server
# other terminals:
make worker
make client
```

`make up` starts an optional Docker Postgres on **5433** if you do not want the system instance.

Bay UI: http://127.0.0.1:5173

Connect **Mock ECU** to exercise the Avensis 3ZR-FAE / Valvematic path without a car. Switch to **OpenPort 2.0 Rev E** when `libopenport.so` is on the laptop (`J2534_LIB` if it is not on the default path).

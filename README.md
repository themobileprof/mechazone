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

First login is the seeded super admin (`admin@mechazone.local` / `change-me-now` unless you set `SUPERADMIN_*`). That account adds shops and issues technician/freelancer logins. Technicians then sign in to use the bay. There is no self-signup.

Connect **Mock ECU** to exercise the Avensis 3ZR-FAE / Valvematic path without a car. Switch to **OpenPort 2.0 Rev E** when `libopenport.so` is on the laptop (`J2534_LIB` if it is not on the default path).

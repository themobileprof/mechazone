# Code map

How the three processes fit together, and where to change them. Product laws: [decisions.md](decisions.md).

```
Browser (React)  --JSON /api-->  Go ledger (:8080)  --SQL-->  Postgres
       |                              |
       |                         UI_DIR = client/dist  (installed)
       '--WebSocket-->  Python worker (:8765)
                            |
                            '-- J2534_LIB  or  udsoncan FakeConnection (mock)
```

Developer loop: `make backend`, `make worker`, `make client` (Vite proxies `/api` and `/healthz` to `:8080`). Installed loop: `scripts/start-mechazone.sh` runs the Go binary and the worker, then opens `http://127.0.0.1:8080`.

---

## Processes

| Process | Entry | Role |
| --- | --- | --- |
| Ledger | `cloud-backend/cmd/server` | Auth, VIN, sessions, imports, playbooks, static UI |
| Worker | `python -m mechazone_worker` | Detect USB, connect kit, identify VIN, deep scan, DID stream |
| Ingest | `cloud-backend/cmd/ingest` | Chunk manuals into Postgres (`make ingest`) |
| UI | `client/src/main.tsx` | Landing / login / Admin / Bay |

---

## Go (`cloud-backend`)

Module path: `mechazone/cloud-backend`. Migrations are embedded (`migrations/*.sql`) and applied on boot. DTC seed: `seeds/p0xxx.csv`.

| Package | Responsibility |
| --- | --- |
| `internal/config` | `.env` from cwd walking up; `IMPORT_DIR`, `LLM_*`, VIN keys |
| `internal/httpapi` | REST + cookie auth + SPA fallback (`ui.go`) |
| `internal/ledger` | Postgres: shops, techs, vehicles, sessions, imports, manuals |
| `internal/vin` | Normalize VIN; vPIC then CarAPI/Vincario |
| `internal/ai` | Playbook fusion, circuit classes, PDF/HTML ingest helpers |
| `internal/importreport` | File sniff, DTC parse, source allowlist, path jail |
| `internal/auth` | Password hash, principals |
| `internal/pii` | Reject customer-identity JSON keys on cloud bodies |

### HTTP (`internal/httpapi`)

Technician cookie required except login, access-request, health, and some decode/DTC/figure reads as marked in `server.go`.

| Method | Path | Handler |
| --- | --- | --- |
| GET | `/healthz` | Process up |
| POST | `/api/v1/auth/login` | Cookie session |
| POST | `/api/v1/auth/logout` | |
| GET | `/api/v1/auth/me` | |
| POST | `/api/v1/access-requests` | Landing ticket |
| GET/POST | `/api/v1/admin/…` | Super admin shops, techs, tickets |
| GET | `/api/v1/vehicles/{vin}` | This shop’s jobs |
| POST | `/api/v1/vehicles/{vin}/decode` | vPIC/cache |
| GET | `/api/v1/dtcs/{code}` | Seeded title + circuit class |
| POST | `/api/v1/sessions` | Live scan ingest (JSON) |
| POST | `/api/v1/vehicles/{vin}/imported-reports` | Multipart attach |
| GET | `/api/v1/sessions/{id}/import` | Shop-scoped file |
| POST | `/api/v1/sessions/{id}/closeout` | Success/fail + parts |
| POST | `/api/v1/playbooks` | LLM fuse |
| GET | `/api/v1/manuals/figures/{id}/image` | Retrieved figure bytes |

Shop/tech IDs are overwritten from `principalFrom` in ingest, import, closeout, and playbook.

Handler files (same package):

| File | Routes |
| --- | --- |
| `server.go` | mux + CORS/logging |
| `auth.go` | login cookie, `requireAuth` / `requireAdmin` / `requireTechnician` |
| `handlers.go` | VIN history, decode, DTC, live session ingest, closeout |
| `import.go` | attach / download scanner files |
| `playbook.go` | LLM fuse |
| `admin.go` / `access.go` | shops, techs, landing tickets |
| `ui.go` | `client/dist` SPA when `UI_DIR` is set |
| `figures.go` | retrieved manual images |

### Postgres (migrations `001`–`007`)

| Table | Role |
| --- | --- |
| `shops`, `technicians`, `users`, `auth_sessions` | Provisioned identity |
| `access_requests` | Landing tickets (no self-signup) |
| `vehicles`, `diagnostic_sessions`, `confirmed_resolutions` | This shop’s job file |
| `session_imports` | Attached scanner files (`007`) |
| `vin_decode_cache` | Forever cache of vPIC/paid decode |
| `dtc_codes` | Seeded SAE P0xxx |
| `doc_sources`, `doc_chunks`, `doc_figures` | Ingested manuals |

---

## Python worker (`diagnostic-worker/mechazone_worker`)

Do not add a second J2534 ctypes layer. `j2534.py` wraps `udsoncan.j2534`. `transport.J2534IsoTpConnection.close()` must **not** PassThru-close the device (one channel, many modules).

| Module | Responsibility |
| --- | --- |
| `__init__.py` | Linux `WINFUNCTYPE` shim |
| `__main__.py` / `ipc.py` | WebSocket JSON commands from the bay |
| `detect.py` | USB VID/PID: OpenPort `0403:cc4d` / `0403:cca2`, ELM detect-only |
| `session.py` | Identify, scan, stream DIDs via `udsoncan.Client` |
| `profiles/` | `select_profile(vin, hints)` — captured map or Toyota/generic probe |
| `j2534.py` | `PassThruOpen(NULL)`, `J2534_LIB` only |
| `hexutil.py` | Hex log lines for the ledger excerpt |

Captured maps live under `profiles/` (`generic_uds.py`, `toyota_common.py`, `avensis_3zr_fae.py`). A timeout is a dark node, not proof the car was built with that ECU.

Mock ECU: `ScriptedEcu` + `FakeConnection`. Default mock VIN is Honda-shaped generic UDS, not Avensis.

---

## TypeScript (`client/src`)

No J2534 imports. Hardware only through `worker.ts`.

| File | Responsibility |
| --- | --- |
| `App.tsx` | Unauthed → Landing; admin → Admin; else Bay |
| `Landing.tsx` / `Login.tsx` | Access request + cookie login |
| `Admin.tsx` | Issue shops and technicians |
| `Bay.tsx` | Kit, VIN, attach report, playbook, jobs, closeout |
| `api.ts` | `fetch` with credentials; FormData must not set `Content-Type` |
| `queue.ts` | `localStorage` JSON queue for session/closeout when offline |
| `types.ts` | Shared DTOs |

---

## Data on disk (shop laptop)

| Path | Contents |
| --- | --- |
| `.env` | Secrets; gitignored |
| `bin/mechazone-server` | Compiled ledger (Release zip or `install.sh`) |
| `client/dist` | Built bay UI |
| `data/imported-reports/{shop-or-tech}/{session}.*` | Attached scanner files |
| `data/manuals/` | PDFs / HTML trees + sidecars (`docs/manuals.md`) |
| `passthru/` | Linux `j2534.so` from pack, or Windows clone DLL (operator) |
| `var/` | `server.log`, `worker.log`, pids |

---

## Tests

```
cd cloud-backend && GOTOOLCHAIN=local go test ./...
cd diagnostic-worker && .venv/bin/python -m pytest tests/ -q
cd client && npx tsc -b
```

CI: `.github/workflows/ci.yml` (test, then `scripts/pack-release.sh`).

# Connecting third-party systems

Shop-floor install (non-technical): **[docs/install.md](install.md)**. This page is for keys, libraries, and compile notes.

Mechazone integrates existing APIs and libraries. Do not rebuild these. Keys stay in environment variables and are never sent to the shop-floor UI.

| System | Role | Cost | Status in this repo |
| --- | --- | --- | --- |
| NHTSA vPIC | First VIN decode | Free | Wired (`VPIC_BASE_URL`) |
| CarAPI | VIN fallback (US-heavy) | Freemium / paid | Wired when `CARAPI_*` set |
| Vincario | VIN fallback (global / EU imports) | Trial then paid | Wired when `VINCARIO_*` set |
| todrobbins/dtcdb | Generic SAE DTC titles | Public CSV | Imported into `dtc_codes` |
| udsoncan + python-can + can-isotp | UDS / ISO-TP | OSS | udsoncan is the scan client; python-can / can-isotp reserved for a later ELM ISO-TP shim |
| NikolaKozina/j2534 | Linux OpenPort 2.0 Pass-Thru | OSS (libusb) | Cloned to `third_party/j2534` — needs one `sudo apt` to compile |
| Tactrix official J2534 | Windows Pass-Thru DLL | Free from Tactrix | Use on Windows only |
| PostgreSQL + pgvector | Ledger + later RAG | OSS | Ledger uses local Postgres; `postgresql-18-pgvector` is the apt package |
| Hosted LLM | Phase 2 playbooks | Paid API | Not called yet |

---

## 1. OpenPort 2.0 / J2534 (the kit)

Official Tactrix drivers and ECUFlash can **erase a clone** (serial blacklist + firmware write). Do not download from [tactrix.com](https://www.tactrix.com/index.php?Itemid=61) onto a machine that will see this cable — internet or not.

On Linux we use the OpenPort-specific OSS library: [NikolaKozina/j2534](https://github.com/NikolaKozina/j2534) (libusb). It is already cloned at `third_party/j2534`. That path never ships Tactrix firmware.

### Windows on an internet-connected laptop (clone-safe)

The bay PC can stay online (VIN decode, ledger). The clone dies when `op20pt32.dll` / ECUFlash **writes firmware**, which can happen offline if a newer official DLL is already on disk.

1. Do **not** run the Tactrix installer, ECUFlash, or “check for updates.”
2. Do **not** let Windows bind the official Tactrix USB driver. If it already did, use Zadig to set **WinUSB** on `0403:cc4d` and `0403:cca2`.
3. Point `J2534_LIB` at a **frozen** PassThru DLL that already matches this clone (seller CD / a copy you keep inside the Mechazone folder). Never replace that file from the internet.
4. Block the updater even if someone later installs it: Windows Firewall outbound deny for `ECUFlash.exe` and any `*tactrix*` binary; hosts file `127.0.0.1` for `tactrix.com` and `www.tactrix.com`.
5. Mechazone must load only `J2534_LIB` (absolute path). Do not rely on the J2534 registry, which may point at an official DLL.

A hosts block alone is not enough: a local official DLL can still flash and brick with no network.

### One-time Linux install (needs sudo)

```bash
sudo apt install libusb-1.0-0-dev pkg-config build-essential postgresql-18-pgvector
cd third_party/j2534/j2534 && make
# optional system install:
# sudo make install
export J2534_LIB="$PWD/j2534.so"

sudo cp deploy/99-openport.rules /etc/udev/rules.d/
sudo udevadm control --reload
sudo usermod -aG dialout "$USER"   # then log out and back in
```

Remove the SD card from the OpenPort before use. Plug the USB cable in, ignition on, then **Refresh kits** — the bay should recommend OpenPort when `0403:cc4d` or `0403:cca2` is present. ELM-class USB is listed as detect-only (no UDS connect). Unknown VID:PID is listed so you can point `J2534_LIB` at another Pass-Thru library. Coverage and capture checklist: `docs/coverage.md`.

USB IDs we allow in `deploy/99-openport.rules`: `0403:cc4d` and `0403:cca2`.

Windows path: set `J2534_LIB` to the frozen clone-matched DLL in the Mechazone folder. Do not install the official Tactrix package.

---

## 2. NHTSA vPIC (default VIN decode)

- Docs: https://vpic.nhtsa.dot.gov/api
- No key. Cache immediately; never re-query a VIN we already stored.
- Endpoint we call: `GET {VPIC_BASE_URL}/vehicles/DecodeVinValues/{vin}?format=json`
- Default: `VPIC_BASE_URL=https://vpic.nhtsa.dot.gov/api`

EU-built Avensis VINs (WMI `SB1`) often come back thin. That is when fallbacks fire.

Test:

```bash
curl -sS "https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/SB1KV56E40E012345?format=json" | head
```

---

## 3. CarAPI (paid / freemium VIN fallback)

- Docs: https://carapi.app/docs/api/vin-decoder/
- Auth docs: https://carapi.app/api?doctype=redoc
- Create a token/secret on the CarAPI dashboard.
- US-sold vehicles are the sweet spot. Free tier hides most years except a test set.

```bash
# 1) JWT (expires ~7 days)
curl -sS -X POST https://carapi.app/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"api_token":"YOUR_TOKEN","api_secret":"YOUR_SECRET"}'

# 2) Decode
curl -sS "https://carapi.app/api/vin/1GTG6CEN0L1139305" \
  -H "Authorization: Bearer YOUR_JWT"
```

Env:

```
CARAPI_TOKEN=
CARAPI_SECRET=
```

Mechazone only calls this after vPIC returns empty, then writes the result into `vin_decode_cache`.

---

## 4. Vincario (global VIN fallback — better for African/EU imports)

- Docs: https://vincario.com/api-docs/3.2/
- Samples: https://github.com/Vincario/api-code-sample
- Trial: https://vincario.com/vin-decoder/ (about 20 free decodes)

Control sum is the first 10 hex chars of:

`sha1("{VIN}|decode|{API_KEY}|{SECRET_KEY}")`

VIN must be uppercase. Secret never goes on the wire.

```
https://api.vincario.com/3.2/{API_KEY}/{CONTROL_SUM}/decode/{VIN}.json
```

Env:

```
VINCARIO_API_KEY=
VINCARIO_SECRET_KEY=
```

Use this for Nigerian/EU imports when vPIC is empty. Still cache-first.

---

## 5. Generic DTC titles (no live web lookup)

Source: [todrobbins/dtcdb generic.csv](https://github.com/todrobbins/dtcdb/blob/master/generic.csv) (public SAE-style P0/U0/C0/B0 titles).

```bash
curl -fsSL -o /tmp/dtcdb-generic.csv \
  https://raw.githubusercontent.com/todrobbins/dtcdb/master/generic.csv
python3 scripts/import_public_dtcs.py /tmp/dtcdb-generic.csv cloud-backend/seeds/p0xxx.csv
# restart the Go server — it upserts the seed on boot
```

Manufacturer codes (Toyota P1xxx, P1047) stay out of this file. Those use VIN history and later RAG.

---

## 6. UDS stack (already in the worker venv)

```
diagnostic-worker/.venv
  udsoncan==1.25.1
  python-can==4.5.0
  can-isotp==2.0.6
  websockets==14.2
```

We wrap `udsoncan` for services. J2534 ISO-TP is the OpenPort path. `python-can` / `can-isotp` are present for later sockets — do not rewrite framing.

---

## 7. PostgreSQL + pgvector

Ledger already uses the Postgres on this machine (`postgres:///mechazone`).

For Phase 2 embeddings:

```bash
sudo apt install postgresql-18-pgvector
psql -d mechazone -c 'CREATE EXTENSION IF NOT EXISTS vector;'
```

Do not add a separate vector-DB product.

---

## 8. Hosted LLM (Phase 2 — do not train a model)

Playbooks call an OpenAI-compatible chat API (DeepSeek is the default). Gemini can be swapped later with a different client. The model sequences tests and lookouts from ledger + network + ingested manuals (any language). It does not draw diagrams. Drop PDFs in `data/manuals/` and run `make ingest` — `docs/manuals.md`. Contract: `docs/playbook.md`. Env:

```
LLM_BASE_URL=
LLM_API_KEY=
LLM_MODEL=
```

Not invoked today.

---

## Environment map

Copy `.env.example`. Only fill keys you actually have.

| Variable | System |
| --- | --- |
| `DATABASE_URL` | Local Postgres |
| `VPIC_BASE_URL` | NHTSA vPIC |
| `CARAPI_TOKEN` / `CARAPI_SECRET` | CarAPI |
| `VINCARIO_API_KEY` / `VINCARIO_SECRET_KEY` | Vincario |
| `J2534_LIB` | Path to `j2534.so` or frozen clone DLL (never from tactrix.com) |
| `LLM_BASE_URL` / `LLM_API_KEY` / `LLM_MODEL` | Hosted playbook model (Phase 2) |
| `MECHAZONE_ADAPTER` | `mock` or `openport2_rev_e` |
| `SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD` | First admin (not a third party) |

---

## What this machine still needs from you (sudo)

This environment cannot `sudo`. Run the apt/udev block in section 1, plug in the OpenPort, then restart `make worker`. VIN fallbacks stay idle until you paste CarAPI or Vincario keys — vPIC keeps working without them.

# Decisions (as built)

`docs/project.md` is the product charter. This file is the **implementation log**: what we chose when building, and what we explicitly refused. If the two disagree, this file wins for the code in this repo.

Decisions are numbered. Do not reverse one without updating this list.

---

## D1 — Software on the kit we already own

**Choice:** OpenPort 2.0 Rev E clone + existing laptop. No ESP32, no second dongle SKU, no issued tablet.

**Why:** The moat is shop-scoped jobs and playbook fusion, not a hardware program. ELM327 is detect-only until someone explicitly asks for a serial fallback.

---

## D2 — Integrate, do not rebuild scanners

**Choice:** Live UDS goes through **udsoncan** (`Client`, `j2534`, `FakeConnection`, `Dtc`, `DidCodec`). Linux only aliases `WINFUNCTYPE` → `CFUNCTYPE` so that module imports.

**Rejected:** A second ctypes PassThru stack; wrapping Autel / Launch / Golo365 / Techstream as the live path; scraping a DTC website; inventing DIDs from a PDF.

**Why:** Independents almost never have ODX. Captured maps are data from a live dump on *this* cable, not a Mechazone OEM database. Avensis is a captured fixture (`MECHAZONE_MOCK_PROFILE` / model hint), not the default VIN.

---

## D3 — Shop file, not a public VIN rap sheet

**Choice:** History is scoped to the login `shop_id`, or to the freelancer `technician_id` when shop is null. Jobs do not follow the VIN to another shop.

**Why:** The product is this workshop’s notebook, not Carfax.

---

## D4 — Provisioned identity; ledger IDs from the session

**Choice:** No self-signup. Super admin creates shops and technicians. `shop_id` / `technician_id` on sessions come from the cookie, never from the JSON body.

**Why:** A bay PC must not let a client payload impersonate another tech.

---

## D5 — Privacy split

**Choice (reversed):** Customer name / phone / plate live on this shop’s ledger (`shop_customers`, keyed by `shop_id`+VIN, or freelancer `technician_id`+VIN). `GET /api/v1/vehicles/{vin}` returns them; `PUT /api/v1/vehicles/{vin}/customer` writes them. Other shops cannot read the row. Playbook fusion sets `shop_work.customer` to nil before the LLM. Session ingest, closeout, and import notes still reject JSON keys like `customer_name`, `phone`, `plate` — identity does not ride on scan payloads. Imported scanner files are not OCR’d.

**Why:** A laptop swap must not lose who owns the car. That is still this shop’s file, not a public VIN rap sheet.

---

## D6 — One Postgres on the bay laptop (for now)

**Choice:** The Go process is the “cloud” API, but the default deploy is **local** Postgres on the shop PC (`postgres:///mechazone`). Offline queue (`client/src/queue.ts`) holds JSON session/closeout posts when `/healthz` is down.

**Not built yet:** A hosted multi-tenant ledger shops sync to; gRPC; Tauri/Electron shell. Customer identity is `shop_customers` on the same Postgres, not a local SQLCipher DB.

**Why:** One laptop can close a job without a public URL. The UI is a browser tab at `127.0.0.1:8080` (installed) or Vite `:5173` (dev).

---

## D7 — VIN decode: vPIC first, cache forever, paid APIs only on miss

**Choice:** NHTSA vPIC (`DecodeVinValues`). On empty/error, CarAPI then Vincario if `CARAPI_*` / `VINCARIO_*` are set. Write `vin_decode_cache` immediately. Never re-query a cached VIN.

**Rejected as product cost:** Keeping Vincario as a monthly requirement for a non-shipping app. Autel/Launch do not buy VIN APIs; they read `$22 F190` and a local coverage file.

**Also not wired:** RapidAPI Europe, Auto.dev, or `CARAPI_API_TOKEN` (wrong name — code reads `CARAPI_TOKEN` / `CARAPI_SECRET`).

---

## D8 — Live scan vs imported report

**Choice:** Live path is OpenPort + worker. Optional **attach scan report** (`adapter_type=imported_report`, `protocol=file_import`) stores a PDF/photo/CSV with typed codes. Not a second live stack. Playbook may use typed codes + shop history; it must gap that this was not an OpenPort capture. Multipart import is **online-only** (not in the JSON offline queue).

---

## D9 — Playbooks: fuse, don’t invent

**Choice:** Hosted LLM (`LLM_*`) after ledger + manual retrieval. Every lookout/cause must cite allowed evidence prefixes. Figures are retrieved, never generated. No `$2F` IO-control IDs until a live capture exists.

**Contract:** `docs/playbook.md`.

---

## D10 — Distribution: compiled folder, not an app store

**Choice:** CI (`make pack` / GitHub Actions) builds `mechazone-linux-amd64.tar.gz` and `mechazone-windows-amd64.zip` with a static Go ledger + `client/dist`. Shop `install.sh` skips Go/Node when that binary is present. Python + Postgres still install on the laptop (worker + job file).

**Rejected:** Public website that scans cars; Android app as the reference host; shipping Tactrix DLLs.

**Windows Pass-Thru:** Operator supplies the clone-matched DLL on CD. Never tactrix.com. Load only `J2534_LIB`.

---

## D11 — Generic DTCs vs manufacturer codes

**Choice:** Seed SAE-style P0xxx from a public CSV into `dtc_codes`. Toyota P1xxx and similar stay for shop history + playbook, not a scraped encyclopedia.

---

## D12 — Clear DTCs is technician-initiated UDS $14

**Choice:** **CLEAR CODES** (Capture and Playbook) calls `udsoncan.Client.clear_dtc`. Group `0xFFFFFF`, then `0x000000` if the ECU NRC-rejects the mask. On session/conditions NRC, one retry in **extended diagnostic session** (`$10 03`) only — never programming session, never security access, never `$2F`. Confirm required. Re-read after. Persist the updated bus capture. Imported reports cannot clear. Dark nodes are not addressed.

**Rejected:** Auto-clear on scan; treating a clear as a repair; seed/key retries; programming session.

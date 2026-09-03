# Mechazone — Product & Architecture Specification

This is the product charter. **What the repo actually does today**, including decisions that diverged from early sketches (Tauri, hosted-only ledger, gRPC), is in [decisions.md](decisions.md). Package-level code map: [code.md](code.md). Doc index: [README.md](README.md).

Mechazone is a shop-floor diagnostic ledger for independent auto technicians. The product is software on a capable Pass-Thru kit we already own: a Tactrix OpenPort 2.0 (Revision E) clone and a laptop. The client reads manufacturer modules on modern ICE vehicles (and later EVs), logs the work done on that vehicle **at this shop**, and uses that shop's job file plus retrieved manuals to close the repair.

The product is not a generic OBD2 scanner, not a dealership OEM laptop, and not a public vehicle-history bureau. A car's jobs stay with the shop that did them so a later seller is not carrying a centralized rap sheet. Spend discipline means integrate OSS, public APIs, and scrape-and-cache sources — not downgrade to a bargain ELM327 that cannot address modern modules or a future EV BMS.

---

## 1. Mission & Value

Independent shops lose hours on modern ICE and Chinese EV electronics because:

- Consumer OBD2 apps only expose emissions PIDs.
- OEM tools and subscriptions are expensive or unavailable.
- A car's prior work lives in another shop's notebook, or nowhere.

Mechazone closes that gap with three assets that compound — none of them a new device we have to buy:

1. **Software on the OpenPort we already have** — J2534 Pass-Thru to the modules that actually fail (ECM, TCU, Valvematic now; BMS and related EV controllers later).
2. **The vehicle ledger** — every scan, freeze-frame, and confirmed fix indexed by VIN.
3. **This shop's job file** — every scan, closeout, and parts list this workshop recorded on that VIN. It does not follow the car to another shop.

Primary market: independent technicians and small shops in Nigeria and the wider African import market, where late-model Toyota ICE platforms and Chinese EVs arrive faster than dealer support.

### Cost and capability laws

- **Software before extra silicon.** If a library, API, or cached scrape can do it reliably, that is the design. Do not buy a second interface to hide a software gap.
- **Capable kit, already owned.** Reference hardware is the OpenPort 2.0 Rev E clone + existing laptop. That class of Pass-Thru is the bar for modern ICE and later EVs (high-speed CAN, UDS, path to CAN-FD). ELM327 v1.5 is a constrained fallback, not the product.
- **No new hardware program.** Do not design ESP32/CAN boards, issue tablets, or add sensors. Do not prescribe gear beyond the OpenPort + laptop already on the bench.
- **Integrate, do not rebuild.** VIN decode, generic DTC text, ISO-TP/UDS/J2534 stacks, PDF parsing, embeddings (`pgvector`), LLM calls, TSB/manual corpora, auth, and shop comms are imported or wrapped. We write the ledger, the OpenPort worker glue, the shop-floor loop, and playbook fusion — that is the moat.

### Core loop

```
OpenPort scan  →  Local ingest  →  this shop's jobs + live telemetry
                                      │
Technician closeout  ←  AI playbook  ←  Community + TSB retrieval
        │
        └──► This shop's job file  (the next visit here starts ahead)
```

1. **Scan** — Technician connects the OpenPort and reads advanced modules, not just Mode $01 PIDs.
2. **Recall** — The client loads that VIN's timeline: prior shops, mileage, DTCs, freeze-frames, parts, verified fixes.
3. **Analyze** — Cloud AI fuses live telemetry, ledger history, and retrieved TSBs/manuals/community resolutions into a shop-floor playbook.
4. **Repair & log** — Technician performs the work and closes the session (success + parts, or actual fix if the playbook failed).
5. **Log the work** — Mechanical facts stay in this shop's file on that VIN. The next visit *here* starts with evidence. Another shop does not inherit it.

Customer identity never leaves the shop. VIN and mechanical facts do.

---

## 2. Reference Kit (already owned)

Development and the capability bar are the hardware already on the bench. There is no custom hardware program and no bargain-dongle redesign.

### 2.1 Surface

| Item | Role |
| --- | --- |
| Existing Windows/Linux laptop | Host — already owned; do not specify a new machine |
| Tactrix OpenPort 2.0 (Revision E) clone | Primary vehicle interface (J2534 Pass-Thru) |
| ELM327 v1.5 (FTDI/CH340), USB or USB-OTG | Constrained fallback only; not the design target |
| Multimeter and hand tools | Assumed shop inventory for playbook pin tests |
| Mechazone client (browser UI + Python J2534 worker) | The product: scan, history, playbook, closeout. As built: React in the browser, not Tauri yet ([D6](decisions.md#d6--one-postgres-on-the-bay-laptop-for-now)) |
| Customer identity on the bay | Names, phones, plates stay in the browser (`localCustomer`). No SQLCipher shop DB yet. |

Do not design ESP32/CAN boards or a field-kit BOM. Rollout is `install.sh` / `install.ps1` (see `docs/install.md`) plus an OpenPort-class J2534 clone (the same class already in hand). Android USB-OTG is a later host option, not the reference.

### 2.2 What software must let a technician do

- Identify the vehicle (VIN from the module or keypad) and pull network history before guessing.
- Address specific modules with UDS (ISO 14229) over ISO-TP (ISO 15765-2).
- Capture active/pending DTCs, freeze-frame, mileage, and raw hex for the session.
- Work through a playbook: pin checks, live parameters, pass/fail criteria.
- Close the job in two taps (or a short actual-fix note) so the network learns.
- Keep working when the shop is offline; sync the mechanical record when the radio returns.

### 2.3 What the product must not require

- Generic SAE J1979-only workflows as the default path.
- Designing as if an ELM327 were enough for modern modules or a later EV BMS.
- A second dongle or custom board because the OpenPort worker was not finished.
- Constant cloud connectivity to complete a scan.
- Customer PII in the cloud.
- Dealership hardware, Windows-only OEM suites, paid VIN APIs on every lookup, or a custom document/helpdesk/ERP we would have to build.

---

## 3. Shop-Floor Workflow

Design every screen and API around this sequence.

```
1. Connect OpenPort → vehicle
2. Read VIN / confirm vehicle
3. Show this shop's jobs  (work done, parts, closeouts — not a public VIN file)
4. Deep module scan   (UDS, not emissions PIDs)
5. Deliver playbook   (automatic after deep scan: faults + live + this shop's jobs + manuals)
6. Technician repairs
7. Closeout           (success + parts  |  fail + actual fix)
8. Sync               (local always; cloud when online)
```

**History-first rule:** if **this shop** already has jobs on the VIN, the first thing the technician sees is that work, not a blank scan form. Other shops do not see this file.

**Closeout rule:** a session is incomplete until the technician records an outcome. Incomplete sessions may stay local; they must not be treated as verified network knowledge.

---

## 4. Users & Network Effects

Identity is provisioned, not open signup.

| Actor | How they get in | Needs from the product |
| --- | --- | --- |
| Super admin | Seeded once (`SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD`) | Create shops; issue technician/freelancer logins |
| Shop technician | Admin creates the account and assigns a shop | Bay scan, this shop's jobs on the VIN, closeout stamped to that shop |
| Freelancer | Admin creates the account with no shop | Same bay; jobs have `technician_id` and a null `shop_id` |
| Next shop on the same VIN | Already a Mechazone shop | Starts a **new** shop file. Does not inherit the previous shop's jobs. |

Ledger writes ignore client-supplied `shop_id` / `technician_id`. The session cookie is the source of truth.

Reputation is a data-quality signal, not a social feed. Verified successful closeouts raise weight. Unverified free text does not outrank a confirmed fix.

---

## 5. Technical Environment

### 5.1 Hardware & vehicles

- **Host (reference):** The existing Windows/Linux laptop. Android USB-OTG is optional later.
- **Interface (reference):** Tactrix OpenPort 2.0 Rev E clone via J2534. ELM327 v1.5 is fallback only.
- **Do not add:** custom CAN hardware, a second prescribed dongle, issued tablets.
- **Vehicles:** Modern ICE first (reference: 2010/2011 Toyota Avensis 3ZR-FAE / Valvematic). Unknown VINs get an ISO 15765-4 UDS probe; Toyota WMIs get an 11-bit probe. Chinese EVs (BYD, GAC, Geely) on high-speed CAN, CAN-FD, and UDS are an explicit later target — keep Pass-Thru/UDS capable of reaching them. What is captured vs still needed: `docs/coverage.md`.

### 5.2 Architectural split

```
┌────────────────────────────────────────────────────────────────────────┐
│                     LOCAL CLIENT (EXISTING LAPTOP)                     │
│  UI (Tauri / TypeScript / Tailwind)  ◄──IPC/WS──►  Python J2534 worker │
│  Encrypted local customer DB          Session queue (offline → cloud)  │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │  gRPC or REST over TLS
                                     │  (when online; otherwise queued)
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          GO CLOUD                                      │
│  Telemetry / history API  →  PostgreSQL VIN ledger + reputation        │
│           │                                                            │
│           ▼                                                            │
│  AI engine (pgvector + existing LLM API) ← imported TSBs/manuals/fixes │
└────────────────────────────────────────────────────────────────────────┘
```

**As built (D6):** the UI is a browser tab; the Go process is the API and optionally serves `client/dist`; Postgres is local on the laptop; JSON REST, not gRPC. Customer name is `localCustomer` in the bay, not SQLCipher. Target sketch above is still the long-term split.

UI never talks to the Pass-Thru library. The Python worker never stores customer PII in payloads destined for the cloud.

---

## 6. System Architecture

### 6.1 Local client

- **UI:** Existing laptop. This shop's jobs on the VIN, module live data, playbook steps, two-click closeout, clear online/offline state.
- **Diagnostic worker (Python 3.11+):** Primary path is J2534 (`openport.dll` / `libopenport.so`) on the OpenPort 2.0 Rev E clone. Wrap maintained OSS for ISO-TP/UDS/CAN (e.g. `udsoncan`, `can-isotp`, `python-can`); own adapter quirks, timeouts, hex validation, and session capture. ELM327 serial is fallback only.
- **IPC:** Local WebSocket or stdin/stdout JSON. Typed contracts only.
- **Local shop partition:** As built: customer name stays in the browser; mechanical rows go to local Postgres (or the JSON offline queue). Spec sketch still allows a later SQLCipher customer DB.

### 6.2 Cloud

- **Gateway (Go):** High-concurrency ingest and VIN-history read APIs. Accepts structured session payloads; rejects PII-looking fields.
- **Network ledger (PostgreSQL):** Shops, technicians, vehicles, sessions, confirmed resolutions, reputation events. Source of truth for "has this VIN been here before?"
- **AI engine:** `pgvector` in the same PostgreSQL + an existing LLM API. Builds playbooks only after ledger + retrieval context is attached. Generic P0xxx definitions come from imported public/OSS seed tables, not from an LLM and not from a live web DTC API.

### 6.3 Session payload (mechanical, cloud-safe)

```json
{
  "vin": "JTDKN3DU5A0123456",
  "shop_id": "uuid",
  "technician_id": "uuid",
  "mileage_km": 142500,
  "adapter_type": "openport2_rev_e",
  "host_os": "linux",
  "protocol": "uds_isotp_can",
  "active_codes": ["P1047", "U011B"],
  "freeze_frame": { "rpm": 1820, "coolant_c": 92, "system_v": 13.8 },
  "raw_hex_stream": ["22 1A 01", "..."],
  "captured_at": "2026-08-15T09:12:00Z"
}
```

VIN must be 17 characters. Adapter, host, and protocol are required so another technician can reproduce the read.

---

## 7. Data Model & Sovereignty

### 7.1 Privacy boundary

| Stays on the shop device | Syncs to the network ledger |
| --- | --- |
| Customer name, phone, plate | VIN, make/model/year (from decode cache) |
| Local job/work-order notes tied to a person | Mileage, DTCs, freeze-frame, raw hex |
| | Confirmed root cause, parts, verification flag |
| | Shop region (country/city), technician id, reputation |

No names, phones, or plates on the shared server.

### 7.2 Relational ledger (PostgreSQL)

```sql
CREATE TABLE shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location_country VARCHAR(100) DEFAULT 'Nigeria',
    location_city VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE technicians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID REFERENCES shops(id) ON DELETE RESTRICT,
    full_name VARCHAR(255) NOT NULL,
    reputation_score INT DEFAULT 100,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vehicles (
    vin VARCHAR(17) PRIMARY KEY,
    make VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    manufacture_year INT NOT NULL,
    decode_source VARCHAR(50), -- vpic | carapi | vincario | manual
    first_seen_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE diagnostic_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vin VARCHAR(17) REFERENCES vehicles(vin) ON DELETE CASCADE,
    shop_id UUID REFERENCES shops(id) ON DELETE RESTRICT,
    technician_id UUID REFERENCES technicians(id) ON DELETE RESTRICT,
    mileage INT NOT NULL,
    adapter_type VARCHAR(64) NOT NULL,
    host_os VARCHAR(32) NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    active_dtc_list TEXT[] NOT NULL,
    freeze_frame_telemetry JSONB,
    raw_hex_excerpt TEXT,
    outcome VARCHAR(16), -- open | success | failed
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE confirmed_resolutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES diagnostic_sessions(id) ON DELETE CASCADE,
    vin VARCHAR(17) REFERENCES vehicles(vin) ON DELETE CASCADE,
    technician_id UUID REFERENCES technicians(id) ON DELETE RESTRICT,
    diagnostic_trouble_code VARCHAR(10) NOT NULL,
    root_cause_explanation TEXT NOT NULL,
    parts_replaced TEXT[] NOT NULL,
    is_verified_fix BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vin_decode_cache (
    vin VARCHAR(17) PRIMARY KEY,
    payload JSONB NOT NULL,
    source VARCHAR(50) NOT NULL,
    cached_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_vin ON diagnostic_sessions (vin, created_at DESC);
CREATE INDEX idx_resolutions_dtc ON confirmed_resolutions (diagnostic_trouble_code);
CREATE INDEX idx_resolutions_verified ON confirmed_resolutions (is_verified_fix)
    WHERE is_verified_fix;
```

### 7.3 Integrate / import / scrape (do not rebuild)

| Need | Source | Our job |
| --- | --- | --- |
| VIN decode | NHTSA vPIC, then CarAPI / Vincario | Call once, write through to `vin_decode_cache` / `vehicles`. Never re-hit a cached VIN. |
| Generic DTC text (P0xxx) | Public/OSS SAE seed lists | Import into PostgreSQL. No per-lookup web DTC APIs. No hand-written encyclopedia. |
| ISO-TP / UDS / CAN / J2534 | `udsoncan` (+ `python-can` / `can-isotp` extras) | Wrap. Linux CFUNCTYPE shim only. Do not reimplement framing. |
| OEM diagnostic description | Licensed ODX/PDX + `odxtools` when a file exists | Do not hand-write a DID encyclopedia. |
| TSB / manual / forum text | Public pages and PDFs | Scrape or bulk-import, cache, chunk with OSS parsers. |
| Embeddings + search | `pgvector` in the same Postgres | No separate vector-DB product. |
| LLM playbooks | Existing hosted LLM API | Prompt + retrieve. Do not train or host a model. |
| Shop comms / peer help | Channels shops already use (e.g. WhatsApp) | Link a session; do not build a helpdesk. |
| Work orders / accounting | Out of scope | Do not build a garage ERP. |

**Cloud AI** is reserved for manufacturer-specific codes (e.g. Toyota P1xxx), deep telemetry, and playbooks that already include this shop's jobs + retrieved manuals.

---

## 8. AI Diagnosis (RAG)

AI is not a chatbot on a DTC. It is the fusion layer that tells a technician **what to do on this vehicle**: which test, in which order, what **this shop already did here**, and — when we have a retrieved figure — how to reach the part.

The model is not allowed to invent a repair, a pin number, or a diagram. It only ranks and sequences evidence we already have: live adapter data, **this shop's jobs on this VIN**, this shop's similar platform closeouts, and retrieved TSB/manual chunks (text + figures).

```
Incoming session (VIN, mileage, DTCs, freeze-frame, live DIDs, adapter)
        │
        ├─► This shop's jobs + closeouts on this VIN
        ├─► This shop's similar jobs (same platform + codes, other vehicles, no VIN)
        ├─► pgvector: imported TSBs/manuals + figure index
        └─► Local P0xxx seed text (if a generic code is present)
        │
        ▼
 Structured prompt  →  Playbook (lookouts, ordered tests, access, cited diagrams, validation)
```

Do not emit a playbook from DTCs alone when this shop already has jobs or manuals. A scan with no DTCs still gets a playbook from live data, the module map, and this shop's jobs. If retrieval has no procedure and no figure, say so and still give adapter tests we can run — do not draw a sketch or quote a pin that was not in a cited chunk.

### 8.1 Prompt pattern

```
[ROLE]
Senior diagnostic engineer. Shop-floor playbooks only. No generic textbook theory.
This exact year / make / model / engine. No “typical Toyota” filler.

[VEHICLE + LIVE ADAPTER DATA]
Identity, powertrain, mileage, active DTCs, freeze-frame, live DIDs, adapter/protocol.

[SHOP JOBS ON THIS VIN — infer lookouts]
This shop's scans and closeouts only: dates, codes, work done, parts.
Call out repeats (same connector, same leak, same code coming back here).

[THIS SHOP'S SIMILAR PLATFORM JOBS]
Same make/model/year-band + codes, other vehicles this shop fixed. No VIN.

[RETRIEVED DOCS + FIGURES]
TSB / manual excerpts. Pin numbers and voltages stay in the same chunk.
Figures (connector views, R&R access, harness routing) only if indexed to this
platform. Each figure is a citation, not a generation.

[DIRECTIVE]
1. Lookouts this technician must not skip, inferred from THIS VIN then the platform.
2. Ordered tests the OpenPort and a shop multimeter can perform, with pass/fail.
3. Access: how to reach the part on THIS body/engine; attach retrieved diagrams only.
4. Validation after the repair. Cite ledger/network/doc IDs. If a fact is missing, say missing.
```

### 8.2 Playbook shape (what the technician sees)

1. **Lookouts for this vehicle** — inferred from **this shop's** jobs before any generic test. Example: this shop already closed a corroded Valvematic connector on this car; inspect that before replacing the actuator.
2. **Probability breakdown** — grounded in ledger + network counts, not vibe.
3. **Ordered tests** — connector/pin/voltage with the shop multimeter, plus live DID/PID the OpenPort worker can read. Pass/fail and where to go next. Do not prescribe extra instruments.
4. **Access on this car** — trim, covers, routing for this year/make/model/engine. If we do not have a retrieved procedure, write “no access procedure on file” instead of inventing one.
5. **Diagrams** — retrieved figures from the same-platform TSB/manual (or a technician photo attached to a verified closeout). Render the cached image with caption + source. **Never** ask the LLM to generate or redraw a wiring diagram.
6. **Validation** — clear codes, warm-up, live-data pass/fail (e.g. Valvematic actual vs target ±0.5°).

Ingestion lives in `cloud-backend/internal/ai/` and uses OSS PDF/HTML parsers. Chunk on semantic boundaries so a pin and its voltage stay in the same passage. Extract and store figures next to the chunk that refers to them (`doc_id`, page, caption, platform key). Playbook JSON cites `figure_id`; the bay loads `/api/v1/docs/figures/{id}`.

Operational contract: `docs/playbook.md`.

---

## 9. Community Feedback Loop

```
Playbook delivered
        │
Technician performs repair
        │
Closeout in the client
        │
   ┌────┴────┐
   ▼         ▼
SUCCESS     FAILED
+reputation  Ask: what actually fixed it?
mark verified index as new edge case
re-weight RAG
```

1. Playbook is shown with this shop's job file, not instead of it.
2. Closeout is mandatory to finish the session: **Fix successful** (parts/actions checklist) or **Fix unsuccessful** (actual fix, short text).
3. Successful verified closeouts stay in this shop's file and inform the next visit here.
4. Failed playbooks plus the real fix update this shop's job — they do not write a public VIN record.
5. Reputation is a data-quality signal for this shop's technicians, not a public vehicle score.

The shop's advantage is accumulated *confirmed* work on the cars it actually saw, not a centralized VIN rap sheet.

---

## 10. Roadmap (Nigeria first)

### Phase 1 — OpenPort worker + ledger (months 1–2)

- Client on the existing laptop. Interface: OpenPort 2.0 Rev E clone via J2534.
- Reference vehicle: 2010/2011 Toyota Avensis 3ZR-FAE. Prove Valvematic and other non-emissions parameters generic OBD2 apps cannot read.
- Wrap OSS ISO-TP/UDS on top of J2534. Local worker → Go ingest → PostgreSQL session + VIN row. Offline queue on the client.
- VIN decode via vPIC with immediate cache. Import a public P0xxx seed; do not write one.

### Phase 2 — History, AI, first shops (months 3–5)

- VIN timeline in the UI before the playbook — **this shop's jobs only**.
- RAG: `pgvector` + existing LLM API over scraped/imported public TSBs/manuals plus this shop's verified closeouts. Playbooks: shop lookouts, ordered adapter/multimeter tests, access on this body/engine, cited figures (never generated).
- Onboard ~10 trusted independent shops with the installer and an OpenPort-class J2534 clone (same class already in hand). No custom boards. ELM327 shops can join later with a reduced module set.
- Keep the worker honest for later Chinese EV work (BMS UDS, CAN-FD path). ICE playbooks ship first.

### Phase 3 — Network product (months 6+)

- Reputation as retrieval weight; badges only as a reflection of that weight.
- Same-VIN revisit at **this shop** opens the prior work immediately. A different shop does not receive that file.
- EV SOH views that use existing live data. Peer help via channels shops already use, linked to a session. No work-order product, no custom helpdesk.

---

## 11. Document roles

| Document | Use |
| --- | --- |
| `.cursorrules` | Binding laws for generated code and architecture choices |
| `docs/project.md` | Product north star: OpenPort software, shop loop, ledger, community, roadmap |

Out of scope: extra hardware we do not already own, custom boards, designing to ELM327 as if it were enough for modern ICE or later EVs, rebuilding VIN/DTC/ISO-TP/J2534/RAG/ERP/helpdesk when OSS or an API exists, this shop's jobs hidden behind a blank scan form, a public VIN rap sheet, customer PII in the cloud, or defaulting to generic OBD2.

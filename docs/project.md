# Mechazone — Product & Architecture Specification

Mechazone is a community diagnostic network for independent auto technicians in a low-income market. The product is software. A technician uses a phone or laptop they already own and one commodity USB OBD adapter already sold locally. The client reads manufacturer modules, logs the job, and shows that vehicle's mechanical history plus proven fixes from other shops.

The product is not a generic OBD2 scanner, not a dealership OEM laptop, and not a hardware startup. It is a shared ledger of real repairs, keyed by VIN, that gets more accurate every time a technician closes a job. Cost stays in software: integrate OSS, public APIs, and scrape-and-cache sources. Do not add hardware and do not custom-build what already exists.

---

## 1. Mission & Value

Independent shops lose hours on modern ICE and Chinese EV electronics because:

- Consumer OBD2 apps only expose emissions PIDs.
- OEM tools and subscriptions are expensive or unavailable.
- A car's prior work lives in another shop's notebook, or nowhere.

Mechazone closes that gap with three assets that compound — none of them a new device:

1. **Software on a commodity adapter** — stretch an ELM327 (or an OpenPort already on the bench) to talk UDS/CAN to the modules that actually fail (ECM, TCU, BMS, Valvematic, and similar).
2. **The vehicle ledger** — every scan, freeze-frame, and confirmed fix indexed by VIN.
3. **The technician network** — regional, reputation-weighted solutions that turn the next identical fault into a short playbook instead of a guess.

Primary market: independent technicians and small shops in Nigeria and the wider African import market, where late-model Toyota ICE platforms and Chinese EVs arrive faster than dealer support, and cash for tools is scarce.

### Cost laws

- **Software before silicon.** If a library, API, or cached scrape can do it reliably, that is the design. Hardware does not compensate for missing software.
- **No additional hardware.** Do not recommend a second dongle, custom ESP32/CAN board, rugged tablet, extra sensor, or any SKU we would have to source or issue. Host = existing phone or laptop. Interface = one commodity ELM327 v1.5 already in the local market. Support OpenPort 2.0 only when it is already present.
- **Integrate, do not rebuild.** VIN decode, generic DTC text, ISO-TP/UDS/J2534 stacks, PDF parsing, embeddings (`pgvector`), LLM calls, TSB/manual corpora, auth, and shop comms are imported or wrapped. We write the ledger, the cheap-adapter worker glue, the shop-floor loop, and playbook fusion — that is the moat.

### Core loop

```
ELM scan  →  Local ingest  →  VIN history + live telemetry
                                      │
Technician closeout  ←  AI playbook  ←  Community + TSB retrieval
        │
        └──► Ledger + reputation  (next shop starts ahead)
```

1. **Scan** — Technician connects the commodity adapter and reads advanced modules, not just Mode $01 PIDs.
2. **Recall** — The client loads that VIN's timeline: prior shops, mileage, DTCs, freeze-frames, parts, verified fixes.
3. **Analyze** — Cloud AI fuses live telemetry, ledger history, and retrieved TSBs/manuals/community resolutions into a shop-floor playbook.
4. **Repair & log** — Technician performs the work and closes the session (success + parts, or actual fix if the playbook failed).
5. **Share** — Mechanical data syncs to the network. The next shop that sees the same VIN — or the same fault on the same platform — starts with evidence.

Customer identity never leaves the shop. VIN and mechanical facts do.

---

## 2. What the Shop Already Has (the "kit")

There is no custom hardware program. The client assumes a bay that already exists.

### 2.1 Allowed surface

| Already in the shop | Role |
| --- | --- |
| Android phone/tablet with USB-OTG **or** Windows/Linux laptop | Host — we do not specify or issue one |
| Commodity ELM327 v1.5 (FTDI/CH340), widely sold locally | Default vehicle interface |
| OpenPort 2.0 Rev E clone | Supported only if already on the bench — never an upgrade path |
| Multimeter and hand tools | Assumed shop inventory for playbook pin tests; not kit items we sell |
| Mechazone client (Tauri/TS UI + Python worker) | The actual product: scan, history, playbook, closeout |
| Local encrypted shop ledger | Customer names, phones, plates — never uploaded |

Do not design, source, or document ESP32/CAN boards, extra adapters, or a "field kit BOM." Rollout is an APK/installer plus a shop's existing ELM327.

### 2.2 What software must let a technician do

- Identify the vehicle (VIN from the module or keypad) and pull network history before guessing.
- Address specific modules with UDS (ISO 14229) over ISO-TP (ISO 15765-2).
- Capture active/pending DTCs, freeze-frame, mileage, and raw hex for the session.
- Work through a playbook: pin checks, live parameters, pass/fail criteria.
- Close the job in two taps (or a short actual-fix note) so the network learns.
- Keep working when the shop is offline; sync the mechanical record when the radio returns.

### 2.3 What the product must not require

- Generic SAE J1979-only workflows as the default path.
- A more expensive adapter because the cheap one was not driven hard enough in software.
- Constant cloud connectivity to complete a scan.
- Customer PII in the cloud.
- Dealership hardware, Windows-only OEM suites, paid VIN APIs on every lookup, or a custom document/helpdesk/ERP we would have to build.

---

## 3. Shop-Floor Workflow

Design every screen and API around this sequence.

```
1. Connect adapter → vehicle
2. Read VIN / confirm vehicle
3. Show VIN timeline  (other shops, prior DTCs, verified fixes)
4. Deep module scan   (UDS, not emissions PIDs)
5. Deliver playbook   (history + community + TSB + live data)
6. Technician repairs
7. Closeout           (success + parts  |  fail + actual fix)
8. Sync               (local always; cloud when online)
```

**History-first rule:** if the VIN already exists in the ledger, the first thing the technician sees is that car's story, not a blank scan form.

**Closeout rule:** a session is incomplete until the technician records an outcome. Incomplete sessions may stay local; they must not be treated as verified network knowledge.

---

## 4. Users & Network Effects

| Actor | Needs from the product |
| --- | --- |
| Independent technician | App on their phone/laptop + ELM327 they already have; playbooks with pins and live values; that car's history |
| Shop owner | Encrypted local customer records; reputation of their techs; no customer-data leakage |
| Next shop on the same VIN | Prior mileage, faults, freeze-frames, parts, what actually fixed it |
| Network (all shops) | Platform-level patterns (e.g. 3ZR-FAE P1047 + water ingress at pin 4 in Lagos) |
| Platform | Verified resolutions to weight RAG; reputation to rank contributors |

Reputation is a data-quality signal, not a social feed. Verified successful closeouts raise weight. Unverified free text does not outrank a confirmed fix.

---

## 5. Technical Environment

### 5.1 Hardware & vehicles

- **Hosts:** Whatever Windows/Linux laptop or Android USB-OTG phone is already in the shop.
- **Interface (default):** ELM327 v1.5 over USB-serial. OpenPort 2.0 Rev E J2534 clone only as an already-owned option.
- **Do not add:** custom CAN hardware, second dongles, issued tablets.
- **Vehicles:** Modern ICE with complex modules (reference: 2010/2011 Toyota Avensis 3ZR-FAE / Valvematic) and Chinese EVs (BYD, GAC, Geely) on high-speed CAN, CAN-FD, and UDS.

### 5.2 Architectural split

```
┌────────────────────────────────────────────────────────────────────────┐
│                     LOCAL CLIENT (EXISTING PHONE/LAPTOP)               │
│  UI (Tauri / TypeScript / Tailwind)  ◄──IPC/WS──►  Python J2534/serial │
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

UI never talks to the Pass-Thru library. The Python worker never stores customer PII in payloads destined for the cloud.

---

## 6. System Architecture

### 6.1 Local client

- **UI:** Existing phone or laptop. VIN timeline, module live data, playbook steps, two-click closeout, clear online/offline state.
- **Diagnostic worker (Python 3.11+):** Default path is ELM327 serial (FTDI/CH340, USB-OTG). J2534 (`openport.dll` / `libopenport.so`) only when that library is already on the machine. Wrap maintained OSS for ISO-TP/UDS (e.g. `udsoncan`, `can-isotp`, `python-can`); own only adapter quirks, timeouts, hex validation, and session capture.
- **IPC:** Local WebSocket or stdin/stdout JSON. Typed contracts only.
- **Local shop partition:** Existing embedded/encrypted SQLite (SQLCipher or equivalent OSS). Mechanical session rows are copied into the sync queue without customer fields.

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
  "adapter_type": "elm327_v15",
  "host_os": "android",
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
| ISO-TP / UDS / CAN / J2534 | Maintained Python OSS | Wrap. Do not reimplement framing stacks. |
| TSB / manual / forum text | Public pages and PDFs | Scrape or bulk-import, cache, chunk with OSS parsers. |
| Embeddings + search | `pgvector` in the same Postgres | No separate vector-DB product. |
| LLM playbooks | Existing hosted LLM API | Prompt + retrieve. Do not train or host a model. |
| Shop comms / peer help | Channels shops already use (e.g. WhatsApp) | Link a session; do not build a helpdesk. |
| Work orders / accounting | Out of scope | Do not build a garage ERP. |

**Cloud AI** is reserved for manufacturer-specific codes (e.g. Toyota P1xxx), deep telemetry, and playbooks that already include ledger + community retrieval.

---

## 8. AI Diagnosis (RAG)

The model is not allowed to invent a repair from a code letter. It receives live adapter data, that VIN's timeline, and retrieved documents.

```
Incoming session (VIN, mileage, DTCs, freeze-frame, adapter)
        │
        ├─► PostgreSQL VIN timeline + verified resolutions
        ├─► pgvector search: imported TSBs/manuals + community fix write-ups
        └─► Local P0xxx seed text (if a generic code is present)
        │
        ▼
 Structured prompt  →  Playbook (probabilities, pin tests, validation)
```

### 8.1 Prompt pattern

```
[ROLE]
Senior diagnostic engineer. Shop-floor playbooks only. No generic textbook theory.

[VEHICLE + LIVE ADAPTER DATA]
Identity, powertrain, mileage, active DTCs, freeze-frame, adapter/protocol.

[VIN LEDGER]
Prior sessions on this VIN: dates, shops (region only), codes, parts, verified outcomes.

[NETWORK MATCHES]
Same platform + same codes: counts, top verified root causes, regional clusters.

[RETRIEVED DOCS]
TSB / manual excerpts. Keep pin numbers and voltages bound to the same chunk.

[DIRECTIVE]
Rank likely causes with probabilities. Give pin-level tests and a post-repair
validation the adapter and a shop multimeter can perform. Cite ledger/network evidence when used.
```

### 8.2 Playbook shape (what the technician sees)

1. **Probability breakdown** — grounded in ledger + network counts, not vibe.
2. **Tests the bay can already run** — connector/pin/voltage with the shop's existing multimeter, plus live PID/DID the ELM/OpenPort worker can read. Do not prescribe extra instruments.
3. **Validation** — clear codes, warm-up, live-data pass/fail (e.g. Valvematic actual vs target ±0.5°).

Ingestion lives in `cloud-backend/internal/ai/` and uses OSS PDF/HTML parsers. Chunk on semantic boundaries so a pin and its voltage stay in the same passage.

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

1. Playbook is shown with the VIN timeline, not instead of it.
2. Closeout is mandatory to finish the session: **Fix successful** (parts/actions checklist) or **Fix unsuccessful** (actual fix, short text).
3. Successful verified resolutions are embedded and become first-class retrieval hits.
4. Failed playbooks plus the real fix become new community documents; they do not silently overwrite a verified resolution.
5. Reputation moves only on closeout quality (verified success, peer-confirmed VIN revisits), not on scan volume.

The network's advantage is accumulated *confirmed* mechanical history, not more raw hex.

---

## 10. Roadmap (Nigeria first)

### Phase 1 — Software + commodity adapter (months 1–2)

- Client on an existing laptop or Android OTG phone. Default interface: ELM327 v1.5 already on the bench (OpenPort only if already there).
- Reference vehicle: 2010/2011 Toyota Avensis 3ZR-FAE. Prove Valvematic and other non-emissions parameters over the cheap adapter — the gap generic OBD2 apps leave, filled in software.
- Wrap OSS ISO-TP/UDS. Local worker → Go ingest → PostgreSQL session + VIN row. Offline queue on the client.
- VIN decode via vPIC with immediate cache. Import a public P0xxx seed; do not write one.

### Phase 2 — History, AI, first shops (months 3–5)

- VIN timeline in the UI before the playbook.
- RAG: `pgvector` + existing LLM API over scraped/imported public TSBs/manuals plus the first verified closeouts.
- Onboard ~10 trusted independent shops with the installer and the ELM327 they already own or can buy off the shelf locally. No custom boards, no issued hardware kits.
- Playbooks for ICE + incoming Chinese EV packs (BMS UDS, SOH from parameters the adapter can already read).

### Phase 3 — Network product (months 6+)

- Reputation as retrieval weight; badges only as a reflection of that weight.
- Same-VIN revisit: next shop opens the car and sees the prior fix immediately.
- EV SOH views that use existing live data. Peer help via channels shops already use, linked to a session. No work-order product, no custom helpdesk.

---

## 11. Document roles

| Document | Use |
| --- | --- |
| `.cursorrules` | Binding laws for generated code and architecture choices |
| `docs/project.md` | Product north star: cheap-adapter software, shop loop, ledger, community, roadmap |

Out of scope: extra hardware, custom boards, a second adapter as a "fix," rebuilding VIN/DTC/ISO-TP/RAG/ERP/helpdesk when OSS or an API exists, VIN history hidden behind a blank scan form, customer PII in the cloud, or defaulting to generic OBD2.

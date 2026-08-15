# Mechazone — Product & Architecture Specification

Mechazone is a community diagnostic network for independent auto technicians. Each technician is issued a small kit. With that kit they read manufacturer modules on modern cars, log what they find, and immediately see that vehicle's mechanical history plus proven fixes from other shops on the network.

The product is not a generic OBD2 scanner and not a dealership OEM laptop. It is a shared ledger of real repairs, keyed by VIN, that gets more accurate every time a technician closes a job.

---

## 1. Mission & Value

Independent shops lose hours on modern ICE and Chinese EV electronics because:

- Consumer OBD2 apps only expose emissions PIDs.
- OEM tools and subscriptions are expensive or unavailable.
- A car's prior work lives in another shop's notebook, or nowhere.

Mechazone closes that gap with three assets that compound:

1. **The kit** — enough hardware and software to talk UDS/CAN to the modules that actually fail (ECM, TCU, BMS, Valvematic, and similar).
2. **The vehicle ledger** — every scan, freeze-frame, and confirmed fix indexed by VIN.
3. **The technician network** — regional, reputation-weighted solutions that turn the next identical fault into a short playbook instead of a guess.

Primary market: independent technicians and small shops in Nigeria and the wider African import market, where late-model Toyota ICE platforms and Chinese EVs arrive faster than dealer support.

### Core loop

```
Kit scan  →  Local ingest  →  VIN history + live telemetry
                                      │
Technician closeout  ←  AI playbook  ←  Community + TSB retrieval
        │
        └──► Ledger + reputation  (next shop starts ahead)
```

1. **Scan** — Technician connects the kit and reads advanced modules, not just Mode $01 PIDs.
2. **Recall** — The client loads that VIN's timeline: prior shops, mileage, DTCs, freeze-frames, parts, verified fixes.
3. **Analyze** — Cloud AI fuses live telemetry, ledger history, and retrieved TSBs/manuals/community resolutions into a shop-floor playbook.
4. **Repair & log** — Technician performs the work and closes the session (success + parts, or actual fix if the playbook failed).
5. **Share** — Mechanical data syncs to the network. The next shop that sees the same VIN — or the same fault on the same platform — starts with evidence.

Customer identity never leaves the shop. VIN and mechanical facts do.

---

## 2. The Technician Kit

The kit is the product surface. Software assumes this kit, not a lab or a dealer SDS subscription.

### 2.1 What ships (v1)

| Item | Role |
| --- | --- |
| Tactrix OpenPort 2.0 (Rev E J2534 clone) **or** ELM327 v1.5 (FTDI/CH340) | Vehicle interface |
| Windows/Linux laptop **or** Android phone/tablet with USB-OTG | Host |
| Mechazone client (Tauri/TS UI + Python diagnostic worker) | Scan, history, playbook, closeout |
| Local encrypted shop ledger | Customer names, phones, plates — never uploaded |

### 2.2 Field kit (v2, community rollout)

Low-cost CAN interfaces (ESP32 + SN65HVD230, sourced locally e.g. Computer Village, Lagos) extend the same client. The protocol stack (CAN / CAN-FD / ISO-TP / UDS) does not change. Adapter type is recorded on every session so history stays reproducible.

### 2.3 What the kit must let a technician do

- Identify the vehicle (VIN from the module or keypad) and pull network history before guessing.
- Address specific modules with UDS (ISO 14229) over ISO-TP (ISO 15765-2).
- Capture active/pending DTCs, freeze-frame, mileage, and raw hex for the session.
- Work through a playbook: pin checks, live parameters, pass/fail criteria.
- Close the job in two taps (or a short actual-fix note) so the network learns.
- Keep working when the shop is offline; sync the mechanical record when the radio returns.

### 2.4 What the kit must not require

- Generic SAE J1979-only workflows as the default path.
- Constant cloud connectivity to complete a scan.
- Customer PII in the cloud.
- Dealership hardware, Windows-only OEM suites, or paid VIN APIs on every lookup.

---

## 3. Shop-Floor Workflow

Design every screen and API around this sequence.

```
1. Connect kit → vehicle
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
| Independent technician | Kit that works on the bay; playbooks with pins and live values; that car's history |
| Shop owner | Encrypted local customer records; reputation of their techs; no customer-data leakage |
| Next shop on the same VIN | Prior mileage, faults, freeze-frames, parts, what actually fixed it |
| Network (all shops) | Platform-level patterns (e.g. 3ZR-FAE P1047 + water ingress at pin 4 in Lagos) |
| Platform | Verified resolutions to weight RAG; reputation to rank contributors |

Reputation is a data-quality signal, not a social feed. Verified successful closeouts raise weight. Unverified free text does not outrank a confirmed fix.

---

## 5. Technical Environment

### 5.1 Hardware & vehicles

- **Hosts:** Windows or Linux laptop; Android USB-OTG.
- **Interfaces:** OpenPort 2.0 Rev E J2534 clone; ELM327 v1.5 over USB-serial; later ESP32+SN65HVD230.
- **Vehicles:** Modern ICE with complex modules (reference: 2010/2011 Toyota Avensis 3ZR-FAE / Valvematic) and Chinese EVs (BYD, GAC, Geely) on high-speed CAN, CAN-FD, and UDS.

### 5.2 Architectural split

```
┌────────────────────────────────────────────────────────────────────────┐
│                        LOCAL CLIENT (THE KIT HOST)                     │
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
│  AI engine (vector DB + LLM RAG)  ←  TSBs, manuals, verified fixes     │
└────────────────────────────────────────────────────────────────────────┘
```

UI never talks to the Pass-Thru library. The Python worker never stores customer PII in payloads destined for the cloud.

---

## 6. System Architecture

### 6.1 Local client

- **UI:** Technician workstation or rugged tablet. VIN timeline, module live data, playbook steps, two-click closeout, clear online/offline state.
- **Diagnostic worker (Python 3.11+):** Loads `openport.dll` / `libopenport.so` via ctypes, or opens the serial ELM/FTDI path. Owns ISO-TP reassembly, UDS request/response, timeouts, and hex validation.
- **IPC:** Local WebSocket or stdin/stdout JSON. Typed contracts only.
- **Local shop partition:** SQLCipher (or equivalent) for customer identity. Mechanical session rows are copied into the sync queue without those fields.

### 6.2 Cloud

- **Gateway (Go):** High-concurrency ingest and VIN-history read APIs. Accepts structured session payloads; rejects PII-looking fields.
- **Network ledger (PostgreSQL):** Shops, technicians, vehicles, sessions, confirmed resolutions, reputation events. Source of truth for "has this VIN been here before?"
- **AI engine:** Builds playbooks only after ledger + retrieval context is attached. Generic P0xxx definitions are served from seed tables, not from an LLM or a public web DTC API.

### 6.3 Session payload (mechanical, cloud-safe)

```json
{
  "vin": "JTDKN3DU5A0123456",
  "shop_id": "uuid",
  "technician_id": "uuid",
  "mileage_km": 142500,
  "adapter_type": "openport2_rev_e",
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

### 7.3 External data rules

- **VIN decode:** NHTSA vPIC first; CarAPI / Vincario as fallback. Write through to `vin_decode_cache` / `vehicles` on first success. Never re-hit an external API for a cached VIN.
- **Generic DTCs (P0xxx):** Local seed tables only. No web DTC lookup APIs.
- **Cloud AI:** Manufacturer-specific codes, deep telemetry, and playbooks that already include ledger + community retrieval.

---

## 8. AI Diagnosis (RAG)

The model is not allowed to invent a repair from a code letter. It receives live kit data, that VIN's timeline, and retrieved documents.

```
Incoming session (VIN, mileage, DTCs, freeze-frame, adapter)
        │
        ├─► PostgreSQL VIN timeline + verified resolutions
        ├─► Vector search: TSBs, OEM manuals, community fix write-ups
        └─► Local P0xxx seed text (if a generic code is present)
        │
        ▼
 Structured prompt  →  Playbook (probabilities, pin tests, validation)
```

### 8.1 Prompt pattern

```
[ROLE]
Senior diagnostic engineer. Shop-floor playbooks only. No generic textbook theory.

[VEHICLE + LIVE KIT DATA]
Identity, powertrain, mileage, active DTCs, freeze-frame, adapter/protocol.

[VIN LEDGER]
Prior sessions on this VIN: dates, shops (region only), codes, parts, verified outcomes.

[NETWORK MATCHES]
Same platform + same codes: counts, top verified root causes, regional clusters.

[RETRIEVED DOCS]
TSB / manual excerpts. Keep pin numbers and voltages bound to the same chunk.

[DIRECTIVE]
Rank likely causes with probabilities. Give pin-level tests and a post-repair
validation that the kit can perform. Cite ledger/network evidence when used.
```

### 8.2 Playbook shape (what the technician sees)

1. **Probability breakdown** — grounded in ledger + network counts, not vibe.
2. **Kit tests** — connector, pin, voltage, live PID/DID the worker can read.
3. **Validation** — clear codes, warm-up, live-data pass/fail (e.g. Valvematic actual vs target ±0.5°).

Vector ingestion lives in `cloud-backend/internal/ai/`. Chunk on semantic boundaries so a pin and its voltage stay in the same passage.

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

### Phase 1 — Kit + ledger prototype (months 1–2)

- One v1 kit: OpenPort 2.0 clone or ELM327 v1.5 on a laptop / Android OTG.
- Reference vehicle: 2010/2011 Toyota Avensis 3ZR-FAE. Prove Valvematic and other non-emissions parameters the kit can read and that generic OBD2 apps cannot.
- Local worker → Go ingest → PostgreSQL session + VIN row. Offline queue on the client.
- VIN decode via vPIC with immediate cache.

### Phase 2 — History, AI, first shops (months 3–5)

- VIN timeline in the UI before the playbook.
- RAG over public TSBs/manuals plus the first verified closeouts.
- Issue ~10 field kits (ESP32 + SN65HVD230 or equivalent) to trusted independent shops.
- Train playbooks on ICE + incoming Chinese EV packs (BMS UDS, SOH-oriented live data).

### Phase 3 — Network product (months 6+)

- Reputation as retrieval weight; badges only as a reflection of that weight.
- Same-VIN revisit: next shop opens the car and sees the prior fix immediately.
- Premium shop tools that still obey the privacy boundary: EV SOH packs, work orders (customer data local), peer help on a specific session.

---

## 11. Document roles

| Document | Use |
| --- | --- |
| `.cursorrules` | Binding laws for generated code and architecture choices |
| `docs/project.md` | Product north star: kit, shop loop, ledger, community, roadmap |

If a change makes the kit harder to use on the bay, drops VIN history behind a scan form, sends customer PII to the cloud, or defaults to generic OBD2, it is out of scope.

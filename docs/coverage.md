# Vehicle and dongle coverage

The bay wraps **udsoncan** (`Client`, `j2534`, `FakeConnection`, `DidCodec`, `Dtc`). It is not a second scanner. Captured maps are data from a live car, not a Mechazone OEM database.

Linux only adds `CFUNCTYPE` so `udsoncan.j2534` can import (upstream uses `WINFUNCTYPE`). Pass-Thru opens `J2534_LIB` with a NULL device name — no Tactrix registry, no firmware IOCTLs.

## In the worker now

| Path | When it runs | What it does |
| --- | --- | --- |
| USB detect | **Refresh kits** | VID/PID for OpenPort `0403:cc4d` / `0403:cca2`, ELM-class chips, unknown USB. |
| OpenPort / J2534 | Connect kit | `udsoncan.j2534` on NikolaKozina/j2534 (Linux) or frozen clone DLL (`J2534_LIB`). |
| Mock ECU | Bench | `udsoncan.FakeConnection`. Generic ISO 15765-4 unless `MECHAZONE_MOCK_PROFILE=<captured id>`. |
| `avensis_3zr_fae` | Decode model Avensis, or the Avensis bench VIN | The only captured map on file. ECM `7E0`, Valvematic `7E2`, DIDs `1A01`–`1A12`. |
| `toyota_common` | Toyota/Lexus WMI, no capture | 11-bit probe. Identity DIDs only. |
| `generic_uds` | Everything else | ISO 15765-4 `7E0`–`7E2`. |
| Tesla / China-EV WMI | listed prefixes | Same probe + an explicit gap. Bring bus type + capture. |

ELM327 is **detect only**. `python-can` / `can-isotp` are udsoncan extras, not a PID reader we wrote.

VIN decode: **vPIC**, then CarAPI / Vincario when keys are set.

## Materials we will not invent — you bring them

Do not ask the worker to grow a scanner. If a fact is missing, it stays in **gaps**. Get the artifact.

### 1. Live OpenPort capture (required for every new platform)

Ignition on, this cable, **Read VIN** + **Deep Scan**. Save:

```
vin
make / model / year (from decode, not a guess)
adapter_type, host_os, protocol
scan JSON (modules, codes, identity, coverage)
raw_hex_stream
which tx/rx answered vs timeout vs NRC
```

Without that file we will not add DIDs, `$2F` IDs, or named modules. A WMI prefix is not a map.

### 2. Licensed diagnostic description (ODX / PDX / CDX / OEM XML)

If you can legally obtain the platform’s ODX (or equivalent), drop it in. We will parse it with **odxtools** (Mercedes-Benz OSS) instead of hand-writing `profiles/`. Until you have the file, do not ask for a DID encyclopedia.

Toyota Techstream / GTS data is not something we scrape. If you have an export you are allowed to use, that is the input.

### 3. Workshop manual / EWD

PDF or HTML tree in `data/manuals/` with a sidecar (`docs/manuals.md`). No sidecar, no ingest. No ingest, no cited pins or figures. The model will not draw a wiring diagram.

### 4. VIN decode keys (EU / Africa / China imports)

vPIC is US-heavy. Set `VINCARIO_*` and/or `CARAPI_*` in `.env`. Cache still wins.

### 5. Frozen Pass-Thru library

- Linux: compile `third_party/j2534` → `J2534_LIB=.../j2534.so`
- Windows: the **clone CD** DLL at an absolute `J2534_LIB`. Never tactrix.com.

### 6. Chinese EV (when you have one on this OpenPort)

Write down DLC bus: CAN 500k ISO-TP vs 29-bit vs **CAN-FD** vs **DoIP**. Captured BMS/VCU tx/rx and a VIN DID that answered. If this clone cannot speak the bus, that is a coverage gap — we do not add a second dongle to hide it.

### 7. Closeout photos (optional, high value)

A connector / loom photo on a **verified** closeout can be shown next to a retrieved figure. It is not a generated diagram.

## What we will not build

- Another UDS/J2534 ctypes stack (udsoncan already has it)
- Autel / Launch / Techstream wrapping
- A DTC website scrape
- Invented Valvematic / BMS IDs for cars we have not captured

## Third-party tools already in use

| Tool | Role |
| --- | --- |
| udsoncan | UDS client, J2534 bindings, FakeConnection, Dtc, DidCodec |
| python-can + can-isotp | udsoncan extras; later ELM ISO-TP shim only |
| NikolaKozina/j2534 | Linux OpenPort Pass-Thru `.so` |
| odxtools | When you have ODX — not wired until a file exists |
| NHTSA vPIC / CarAPI / Vincario | VIN decode |
| todrobbins/dtcdb | Generic P0xxx titles |
| Hosted LLM | Playbook fusion |

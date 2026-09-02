# Vehicle and dongle coverage

What the bay can do today, which **third-party tools** it already wraps, and what we still need from a live shop (captures, manuals, cable IDs) before claiming a platform or an EV.

The product does not launch other scanner apps. It keeps one session (identify → scan → playbook → closeout) and plugs OSS under it.

## In the worker now

| Path | When it runs | What it does |
| --- | --- | --- |
| USB detect | **Refresh kits** | VID/PID for OpenPort `0403:cc4d` / `0403:cca2`, ELM-class chips, unknown USB. Recommends OpenPort when that cable is present. |
| OpenPort / J2534 | Connect kit | `udsoncan` + NikolaKozina/j2534 (Linux) or frozen clone DLL (`J2534_LIB`). UDS over ISO-TP, 11-bit, 500 kbit. |
| Mock ECU | Bench | Generic ISO 15765-4 (Honda-shaped VIN). Pin a captured map with `MECHAZONE_MOCK_PROFILE=<id>` (e.g. `avensis_3zr_fae`). |
| `avensis_3zr_fae` | Decode model Avensis, or the Avensis bench VIN | Captured ECM `7E0`, Valvematic `7E2`, Toyota 11-bit probes, live DIDs `1A01`–`1A12`. Not selected from Toyota WMI / `SB1` alone. |
| `toyota_common` | Toyota/Lexus WMI, no captured platform | Same 11-bit **probe**. No family-specific live DIDs. Identity DIDs only (`F187`, `F18A`, `F18C`). |
| `generic_uds` | Everything else | ISO 15765-4 physical `7E0`–`7E2`: VIN `F190`, DTCs `$19`, ISO identity DIDs. No OEM body map, no invented live scales. |
| Tesla / China-EV WMI | `5YJ`/`7SA`/`LRW` or listed BYD/GAC/Geely/Chery prefixes | Still `generic_uds`, plus an explicit BMS/proprietary **gap**. |

ELM327 is **detected only**. Connecting it is refused. `python-can` and `can-isotp` are already in the worker venv for a later thin serial fallback — not SAE J1979 as the default scan.

VIN decode stays **vPIC**, then CarAPI / Vincario when those keys are set (`docs/integrations.md`).

## What you need to fill

Do not invent addresses. A new platform lands when we have a **live OpenPort capture** on a car you can put in the bay, plus manuals if you have them.

### 1. ICE platforms you actually see (first)

For each make/model/year band (Honda, later Toyota, Nissan, Hyundai — whatever is on the ramp):

1. Ignition on, OpenPort connected, **Read VIN** + **Deep Scan**. Save the worker hex (`raw_hex_stream`) and the JSON scan.
2. Note which modules answered (tx/rx, NRC vs timeout).
3. If you have a workshop manual or EWD, drop it in `data/manuals/` with a sidecar and `make ingest` (`docs/manuals.md`).
4. Optional: one DID list you **saw** (service `$22` responses), not a guess. `$2F` IO-control IDs only if the capture shows them.

That becomes a new file under `diagnostic-worker/mechazone_worker/profiles/` and an entry in `CAPTURED` (`profiles/__init__.py`). Until then the car gets the generic or Toyota probe and the playbook says so.

### 2. Other J2534 cables (not bargain ELM)

If a shop has another Pass-Thru stick (DrewTech, Mongoose, …):

- USB **VID:PID** from `Refresh kits` (unknown USB line) or `lsusb`.
- Path to **that** vendor’s J2534 library. Set `J2534_LIB`. Never download `op20pt32.dll` from tactrix.com for this clone.

We can then treat it as another `uds_j2534` device. We will not wrap Autel/Launch tablet apps or Bluetooth ELM clones as the design path.

### 3. ELM327 fallback (only if a shop has nothing else)

Need a **maintained** ISO-TP-over-ELM path (we already depend on `python-can` / `can-isotp`). Still a degraded session: record `adapter_type=elm327`, no pretend Valvematic/BMS. Do not turn the product into a PID reader.

### 4. Chinese EVs (BYD, GAC, Geely, Chery, …)

Need, on a car you can reach with **this** OpenPort:

- Whether the DLC is classic CAN 500k ISO-TP, **29-bit**, **CAN-FD**, or **DoIP** (Ethernet). This clone is J2534-1 CAN. If the car only speaks CAN-FD/DoIP on the DLC, write that down — we do not add a second dongle to hide it.
- Captured BMS / VCU / inverter **tx/rx** and a VIN DID that actually answered.
- Any public or licensed manual we can ingest (same scrape-and-cache rules).

WMI prefixes in `profiles/__init__.py` only **flag** the gap. They are not a diagnosis.

### 5. American EVs (Tesla, some GM/Ford)

- Tesla: OBD UDS is usually not a service path. Need a captured map or we keep the Tesla gap. Do not wrap Tesla Toolbox.
- Ford/GM: same as (4) — bus type + UDS addresses from a live Pass-Thru session, plus security/gateway notes. No invented IDs.

### 6. VIN decode for African / China VINs

vPIC is US-heavy. For EU/Africa/China imports, **Vincario** (or CarAPI) keys in `.env` are the fill. Cache still wins; we never re-query a stored VIN.

## Capture bundle (send this, not a guess)

For one vehicle:

```
vin
make / model / year (if known)
adapter_type, host_os, protocol
scan JSON (modules, codes, identity, coverage)
raw_hex_stream
optional: photo of the DLC / cable, lsusb line
optional: manual PDF or HTML tree for ingest
```

That is enough to add a profile without inventing pins.

## Third-party tools already in use

| Tool | Role |
| --- | --- |
| udsoncan | UDS client |
| python-can + can-isotp | In venv; reserved for a later ELM ISO-TP shim |
| NikolaKozina/j2534 | Linux OpenPort Pass-Thru |
| NHTSA vPIC / CarAPI / Vincario | VIN decode |
| todrobbins/dtcdb | Generic P0xxx titles |
| Hosted LLM | Playbook fusion (not a code encyclopedia) |

Do not add a second scanner GUI. Do not scrape a DTC website per lookup.

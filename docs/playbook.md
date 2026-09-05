# Shop-floor playbook (AI fusion)

Mechazone AI exists to put a technician on the next correct action for **this vehicle**: what to test, what history says to watch, how to reach the part, and a cited diagram when we have one.

It is not a chat box on a fault code. It does not invent pins, voltages, or drawings.

## Inputs (all required before a full playbook)

| Input | Source | If missing |
| --- | --- | --- |
| Live scan | OpenPort worker (VIN, DTCs, freeze-frame, DIDs, module map, adapter). Profile is a captured platform map when we have one, otherwise Toyota 11-bit probe or ISO 15765-4. | Adapter tests only; no “typical car” essay. No DTCs is still a scan — advise from live + modules + shop jobs. Coverage gaps stay visible. |
| Imported report (optional) | File attached on the bay (`adapter_type=imported_report`). Typed codes + note only — the PDF is not sent to the model. | Not a live capture. Gap: confirm with this OpenPort. |
| This shop's jobs on this VIN | PostgreSQL sessions + closeouts scoped to the login shop (or freelancer) | Still retrieve manuals; say first visit to this shop |
| This shop's similar platform jobs | Same make/model/year-band + codes, **this shop only**, no other VIN | Omit that section |
| Retrieved docs | Manual corpus (`data/manuals` ingest). Hybrid FTS + `pgvector` cosine on the same Postgres, still scoped to this platform or a pinned book. Licensed ODX/PDX (odxtools) if you have it. | No pin numbers that were not retrieved. Chunks may be in any language. |
| Retrieved figures | Figures indexed to those chunks / platform key | Show “no diagram on file” — do not generate one |

Do not call the LLM until the shop-job lookup and retrieval queries have run. A scan with no DTCs still gets a playbook from live data, the module map, and this shop's jobs. Generic P0xxx seed text may attach; it must not replace this shop's work log. A vehicle's jobs do not follow it to another shop.

## Output the bay renders

Example for a vehicle that matched a captured platform map (here, Avensis). The same shape applies to any VIN; lookouts and steps come from this shop's jobs and retrieved docs, not from a hardcoded car.

```json
{
  "vin": "SB1KV56E40E012345",
  "platform": "toyota avensis 2009-2012 3zr-fae",
  "lookouts": [
    {
      "text": "This shop already closed P1047 on this car as a corroded Valvematic connector. Inspect that connector before replacing the actuator.",
      "evidence": ["resolution:<id>"]
    }
  ],
  "likely_causes": [
    {
      "title": "Valvematic connector / circuit",
      "probability": 0.62,
      "evidence": ["ledger", "network:12"]
    }
  ],
  "steps": [
    {
      "order": 1,
      "kind": "test",
      "title": "Compare target vs actual angle",
      "detail": "UDS on ECM 7E0, DIDs 1A01 / 1A02.",
      "pass": "Within 0.5° after warm idle.",
      "fail": "Go to step 2.",
      "adapter": true
    },
    {
      "order": 2,
      "kind": "access",
      "title": "Reach the Valvematic actuator on this engine",
      "detail": "Cited procedure from retrieved manual chunk only.",
      "figures": ["figure:<id>"]
    }
  ],
  "validation": "Clear codes, road test, actual vs target still within 0.5°.",
  "gaps": ["No access figure on file for this body."]
}
```

Every lookout, pin, voltage, and figure must cite `evidence` or `figures`. If the model cannot cite it, it goes in `gaps`, not in `steps`.

## Bay checks (iterate the playbook)

Playbook steps become shop-scoped checks on this VIN (`playbook_checks`). They are not a job closeout.

On the bay, stamp **DID THIS** (you ran the test / did the correction) or **NOT THIS** (you are sure it is not the fault). Optional finding note — no name, phone, or plate. Rebuild sends settled checks as `bay_checks` so the next playbook can move to the next test instead of repeating a dead end.

A ruled-out step stays ruled out across rebuilds. New steps from the next playbook open as new checks. Other shops cannot read this row.

## How-to cards (shop skills, not this VIN)

Playbook steps that mention ohms, volts, continuity, the 16-pin DLC, or backprobing show a **HOW-TO** button. The card teaches leads, dial, and the display. It does not invent pin numbers on a module connector. DLC pins 16 / 4 / 5 are SAE J1962 only. Plates live in `client/public/howto/` (`docs` hunt list in that folder’s README).

## Diagrams

- Source: OSS-parsed TSB/manual PDFs (figure objects) or a photo a technician attached to a **verified** closeout.
- Store: `doc_figures` (id, doc_id, page, caption, platform key, image bytes or object path).
- Show: bay iframe/img from our API, caption, document name. Not a model-drawn SVG.
- YouTube embeds are a later optional aid (`docs` video matching). They do not replace a cited workshop figure.

## History inference (this shop's work)

Before ranking causes, walk **this shop's** jobs on this VIN:

- Same code coming back after a closeout here → do not repeat that part swap first.
- Prior connector / water ingress / loom repair **at this shop** → lookout on that area.
- Parts this shop already replaced → do not lead with that part unless live data contradicts.

Then this shop's similar jobs on the same platform (other vehicles, no VIN in the prompt). Unverified narrative never outranks a confirmed closeout. Do not load another shop's file on this VIN. Do not send customer name, phone, or plate to the model (`shop_work.customer` is stripped in `fuse.go`).

## Wiring and dark nodes

Circuit / U-codes are classified (open, short-to-batt, short-to-gnd, lost communication, bus-off) before the model runs. Retrieval then prefers EWD and connector figures already ingested. The worker probes confirmed powertrain nodes plus Toyota 11-bit addresses; a timeout is a dark node, an NRC still counts as on the bus.

A DID wiggle log streams ECM identifiers while the technician moves the loom. There are no captured UDS `$2F` IO-control IDs on any profile yet — do not invent them.

## Implementation home

`cloud-backend/internal/ai/` — retrieve (FTS + optional cosine), cite, call an existing LLM API (`LLM_*` in `.env`). Chunk embeddings are always local Ollama `bge-small-en-v1.5`. Do not train or host a playbook model. Do not rebuild a PDF product; wrap an OSS parser.

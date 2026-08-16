# Shop-floor playbook (AI fusion)

Mechazone AI exists to put a technician on the next correct action for **this vehicle**: what to test, what history says to watch, how to reach the part, and a cited diagram when we have one.

It is not a chat box on a fault code. It does not invent pins, voltages, or drawings.

## Inputs (all required before a full playbook)

| Input | Source | If missing |
| --- | --- | --- |
| Live scan | OpenPort worker (VIN, DTCs, freeze-frame, DIDs, adapter) | Adapter tests only; no “typical car” essay |
| This VIN ledger | PostgreSQL timeline + closeouts | Still retrieve platform + docs; say first-seen |
| Network matches | Same make/model/engine/year-band + codes, reputation-weighted | Omit network section |
| Retrieved docs | `pgvector` over imported TSB/manual/community chunks | No pin numbers that were not retrieved |
| Retrieved figures | Figures indexed to those chunks / platform key | Show “no diagram on file” — do not generate one |

Do not call the LLM until the ledger lookup and retrieval queries have run. Generic P0xxx seed text may attach; it must not replace history.

## Output the bay renders

```json
{
  "vin": "SB1KV56E40E012345",
  "platform": "toyota avensis 2009-2012 3zr-fae",
  "lookouts": [
    {
      "text": "This VIN had P1047 closed as a corroded Valvematic connector. Inspect that connector before replacing the actuator.",
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

## Diagrams

- Source: OSS-parsed TSB/manual PDFs (figure objects) or a photo a technician attached to a **verified** closeout.
- Store: `doc_figures` (id, doc_id, page, caption, platform key, image bytes or object path).
- Show: bay iframe/img from our API, caption, document name. Not a model-drawn SVG.
- YouTube embeds are a later optional aid (`docs` video matching). They do not replace a cited workshop figure.

## History inference (this is the moat)

Before ranking causes, walk this VIN:

- Same code coming back after a “fix” → do not repeat that part swap first.
- Prior connector / water ingress / loom repair → lookout on that area.
- Parts already replaced → do not lead with that part unless live data contradicts.

Then the platform network. Unverified narrative never outranks a confirmed resolution.

## Implementation home

`cloud-backend/internal/ai/` — retrieve, cite, call an existing LLM API (`LLM_*` in `.env`). Do not train or host a model. Do not rebuild a PDF product; wrap an OSS parser.

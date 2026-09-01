"""Classify DTCs as circuit / bus vs component. Does not invent pin data."""

from __future__ import annotations

from typing import Any


def classify_code(code: str, title: str = "") -> dict[str, str]:
    code = (code or "").strip().upper()
    title = (title or "").strip().lower()
    out = {"code": code, "class": "component", "reason": "No circuit or bus pattern in the code or title."}
    if not code:
        return out
    if code.startswith("U"):
        if "0073" in code or "bus off" in title:
            return {"code": code, "class": "bus_off", "reason": "Network bus-off / U0073-class."}
        return {"code": code, "class": "lost_communication", "reason": "U-code: lost communication with a module."}
    if any(s in title for s in ("short to battery", "short to batt", "short to b+", "circuit high")):
        return {"code": code, "class": "short_to_battery", "reason": "Title is circuit-high / short to battery."}
    if any(s in title for s in ("short to ground", "short to gnd", "circuit low")):
        return {"code": code, "class": "short_to_ground", "reason": "Title is circuit-low / short to ground."}
    if any(s in title for s in ("open circuit", "circuit open", "open in")):
        return {"code": code, "class": "open_circuit", "reason": "Title is an open circuit."}
    if "circuit" in title:
        return {"code": code, "class": "circuit", "reason": "Title names a circuit, not only a component."}
    return out


def classify_codes(codes: list[str], titles: dict[str, str] | None = None) -> list[dict[str, str]]:
    titles = titles or {}
    return [classify_code(c, titles.get(c.upper(), titles.get(c, ""))) for c in codes]


def wiring_shaped(classes: list[dict[str, str]]) -> bool:
    return any(
        row["class"] in {
            "open_circuit",
            "short_to_battery",
            "short_to_ground",
            "circuit",
            "lost_communication",
            "bus_off",
        }
        for row in classes
    )


def network_hint(modules: list[dict[str, Any]]) -> dict[str, Any]:
    live = [m for m in modules if m.get("reachable")]
    dark = [m for m in modules if not m.get("reachable")]
    ecm_up = any(m.get("name") == "ECM" and m.get("reachable") for m in live)
    confirmed_dark = [str(m.get("name")) for m in dark if m.get("confirmed")]
    hint: dict[str, Any] = {"live": len(live), "dark": len(dark)}
    if not ecm_up:
        hint["reading"] = "backbone"
        hint["summary"] = "ECM did not answer. Check DLC power, ground, and CAN before blaming a single module."
        return hint
    if confirmed_dark:
        hint["reading"] = "branch"
        hint["summary"] = (
            "ECM is live. Silent confirmed node: "
            + ", ".join(confirmed_dark)
            + ". That branch (power / ground / CAN), not the whole bus."
        )
        return hint
    if len(dark) >= 3:
        hint["reading"] = "probes_silent"
        hint["summary"] = (
            "ECM is live. Extra Toyota 11-bit probes did not answer — they may be absent on this car. "
            "Do not treat as a backbone failure."
        )
        return hint
    hint["reading"] = "ok"
    hint["summary"] = "Probed modules answered."
    return hint

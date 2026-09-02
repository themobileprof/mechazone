"""Toyota/Lexus 11-bit probe when the VIN is Toyota but not a captured platform.

ECM 7E0 is ISO 15765-4. Other addresses are common on this era — a timeout is
dark, not proof the car was built with that ECU. No Valvematic DIDs unless the
Avensis profile matched.
"""

from __future__ import annotations

from mechazone_worker.profiles.base import Module, VehicleProfile

ECM = Module("ECM", 0x7E0, 0x7E8, "powertrain", confirmed=True)
TCM = Module("TCM", 0x7E1, 0x7E9, "powertrain", confirmed=False)
ECU_7E2 = Module("ECU_7E2", 0x7E2, 0x7EA, "powertrain", confirmed=False)
ABS = Module("ABS", 0x7B0, 0x7B8, "chassis", confirmed=False)
METER = Module("METER", 0x7C0, 0x7C8, "body", confirmed=False)
BODY = Module("BODY", 0x750, 0x758, "body", confirmed=False)
SRS = Module("SRS", 0x780, 0x788, "body", confirmed=False)

PROFILE = VehicleProfile(
    id="toyota_common",
    make="Toyota",
    model="",
    year=0,
    modules=(ECM, TCM, ECU_7E2, ABS, METER, BODY, SRS),
    dids=(),
    depth="toyota_probe",
    gaps=(
        "Toyota 11-bit probe only — not a captured platform map.",
        "No manufacturer live DIDs (Valvematic / family-specific) until a live OpenPort capture.",
        "No $2F IO-control IDs on file.",
    ),
)

"""ISO 15765-4 UDS probe when we have no captured OEM map.

Reads VIN (F190), DTCs ($19), and ISO identity DIDs. Does not invent
manufacturer live values, body addresses, BMS IDs, or IO-control.
"""

from __future__ import annotations

from mechazone_worker.profiles.base import ISO15765_4_MODULES, VehicleProfile

PROFILE = VehicleProfile(
    id="generic_uds",
    make="",
    model="",
    year=0,
    modules=ISO15765_4_MODULES,
    dids=(),
    depth="iso_15765_4",
    gaps=(
        "ISO 15765-4 physical only (7E0–7E2). No OEM body/chassis map.",
        "Bring a live OpenPort hex dump + scan JSON from this car, or a licensed ODX/PDX for odxtools. Do not invent DIDs.",
        "Workshop manual / EWD in data/manuals/ with a sidecar, or the playbook has no pins or figures.",
        "No $2F IO-control IDs until a capture shows them.",
    ),
)

# Honda-shaped bench VIN for tests — not a captured Honda map.
GENERIC_MOCK_VIN = "1HGCM82633A004352"


def mock_replies(tx_id: int, vin: str = GENERIC_MOCK_VIN) -> dict[bytes, bytes] | None:
    if tx_id != 0x7E0:
        return None
    vin_ascii = vin.encode("ascii")[:17].ljust(17, b" ")
    return {
        bytes([0x22, 0xF1, 0x90]): bytes([0x62, 0xF1, 0x90]) + vin_ascii,
        bytes([0x19, 0x02, 0xFF]): bytes([0x59, 0x02, 0xFF, 0x01, 0x13, 0x00, 0x2F]),
        bytes([0x22, 0xF1, 0x87]): bytes([0x62, 0xF1, 0x87]) + b"ECM-HONDA-DUMMY",
    }

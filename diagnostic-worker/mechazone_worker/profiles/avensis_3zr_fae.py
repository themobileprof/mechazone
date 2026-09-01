"""2010/2011 Toyota Avensis 3ZR-FAE / Valvematic — UDS module map.

ECM 7E0 and Valvematic 7E2 are the confirmed powertrain nodes for this profile.
Other 11-bit addresses are probes only: ISO 15765-4 (TCM) or addresses Toyota
commonly uses on this era. A timeout means dark — not proof the car was built
with that ECU. Do not invent IO-control / routine IDs; none are captured yet.
"""

from __future__ import annotations

from dataclasses import dataclass


MOCK_VIN = "SB1KV56E40E012345"


@dataclass(frozen=True)
class Module:
    name: str
    tx_id: int
    rx_id: int
    family: str
    confirmed: bool = False


@dataclass(frozen=True)
class DataIdentifier:
    did: int
    name: str
    unit: str
    scale: float
    offset: float = 0.0
    size: int = 2


PROFILE_ID = "avensis_3zr_fae"
MAKE = "Toyota"
MODEL = "Avensis"
YEAR = 2011
POWERTRAIN = "3ZR-FAE Valvematic"

ECM = Module("ECM", 0x7E0, 0x7E8, "powertrain", confirmed=True)
VALVEMATIC = Module("VALVEMATIC", 0x7E2, 0x7EA, "powertrain", confirmed=True)
TCM = Module("TCM", 0x7E1, 0x7E9, "powertrain", confirmed=False)  # ISO 15765-4
ABS = Module("ABS", 0x7B0, 0x7B8, "chassis", confirmed=False)  # Toyota 11-bit common
METER = Module("METER", 0x7C0, 0x7C8, "body", confirmed=False)
BODY = Module("BODY", 0x750, 0x758, "body", confirmed=False)
SRS = Module("SRS", 0x780, 0x788, "body", confirmed=False)

MODULES = (ECM, VALVEMATIC, TCM, ABS, METER, BODY, SRS)

# Captured IOControl (UDS $2F) IDs go here after a live OpenPort capture. Empty on purpose.
IO_CONTROLS: tuple[int, ...] = ()

VIN_DID = 0xF190

# Candidate manufacturer DIDs — confirm with a live OpenPort capture.
DIDS = (
    DataIdentifier(0x1A01, "valvematic_target_angle", "deg", 0.1),
    DataIdentifier(0x1A02, "valvematic_actual_angle", "deg", 0.1),
    DataIdentifier(0x1A10, "engine_rpm", "rpm", 1.0),
    DataIdentifier(0x1A11, "coolant_temp", "C", 1.0, offset=-40, size=1),
    DataIdentifier(0x1A12, "system_voltage", "V", 0.1),
)


def _u16(n: int) -> bytes:
    return int(n).to_bytes(2, "big")


def mock_replies(tx_id: int) -> dict[bytes, bytes] | None:
    """Scripted ECU answers. None = silent (dark node)."""
    if tx_id != ECM.tx_id:
        return None
    vin_ascii = MOCK_VIN.encode("ascii")
    return {
        bytes([0x22, 0xF1, 0x90]): bytes([0x62, 0xF1, 0x90]) + vin_ascii,
        bytes([0x19, 0x02, 0xFF]): bytes([0x59, 0x02, 0xFF, 0x10, 0x47, 0x00, 0x2F, 0xC1, 0x1B, 0x00, 0x2F]),
        bytes([0x22, 0x1A, 0x01]): bytes([0x62, 0x1A, 0x01]) + _u16(125),  # 12.5 deg
        bytes([0x22, 0x1A, 0x02]): bytes([0x62, 0x1A, 0x02]) + _u16(0),
        bytes([0x22, 0x1A, 0x10]): bytes([0x62, 0x1A, 0x10]) + _u16(1820),
        bytes([0x22, 0x1A, 0x11]): bytes([0x62, 0x1A, 0x11, 132]),  # 92 C
        bytes([0x22, 0x1A, 0x12]): bytes([0x62, 0x1A, 0x12]) + _u16(138),  # 13.8 V
    }

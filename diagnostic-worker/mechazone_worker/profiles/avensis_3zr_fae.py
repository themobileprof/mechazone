"""2010/2011 Toyota Avensis 3ZR-FAE / Valvematic — UDS module map.

DIDs marked candidate_* are captured or community-reported and must be
confirmed on the physical car. VIN and DTC services are ISO 14229 standard.
"""

from __future__ import annotations

from dataclasses import dataclass

MOCK_VIN = "SB1KV56E40E012345"


@dataclass(frozen=True)
class Module:
    name: str
    tx_id: int
    rx_id: int


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

ECM = Module("ECM", 0x7E0, 0x7E8)

# Valvematic actuator is a separate node on some T27 cars (U011B).
# When the actuator is silent, reads still go to the ECM for shared sensors.
VALVEMATIC = Module("VALVEMATIC", 0x7E2, 0x7EA)

MODULES = (ECM, VALVEMATIC)

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


def mock_replies(tx_id: int) -> dict[bytes, bytes]:
    """Scripted ECU answers for bench development without a vehicle."""
    vin_ascii = MOCK_VIN.encode("ascii")
    replies: dict[bytes, bytes] = {
        bytes([0x22, 0xF1, 0x90]): bytes([0x62, 0xF1, 0x90]) + vin_ascii,
        # Report DTCs by status mask — P1047 + U011B on ECM
        bytes([0x19, 0x02, 0xFF]): bytes([0x59, 0x02, 0xFF, 0x10, 0x47, 0x00, 0x2F, 0xC1, 0x1B, 0x00, 0x2F]),
        bytes([0x22, 0x1A, 0x01]): bytes([0x62, 0x1A, 0x01]) + _u16(125),  # 12.5 deg
        bytes([0x22, 0x1A, 0x02]): bytes([0x62, 0x1A, 0x02]) + _u16(0),
        bytes([0x22, 0x1A, 0x10]): bytes([0x62, 0x1A, 0x10]) + _u16(1820),
        bytes([0x22, 0x1A, 0x11]): bytes([0x62, 0x1A, 0x11, 132]),  # 92 C
        bytes([0x22, 0x1A, 0x12]): bytes([0x62, 0x1A, 0x12]) + _u16(138),  # 13.8 V
    }
    if tx_id == VALVEMATIC.tx_id:
        # Actuator offline — NRC 0x10 generalReject / no communication analogue
        return {
            bytes([0x22, 0xF1, 0x90]): bytes([0x7F, 0x22, 0x10]),
            bytes([0x19, 0x02, 0xFF]): bytes([0x7F, 0x19, 0x10]),
        }
    return replies

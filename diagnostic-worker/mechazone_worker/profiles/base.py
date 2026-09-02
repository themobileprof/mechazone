"""Shared UDS profile types. Do not invent pins, $2F IDs, or manufacturer DIDs here."""

from __future__ import annotations

from dataclasses import dataclass, field

from udsoncan import DataIdentifier as IsoDataIdentifier


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


# ISO 14229-1 identity DIDs — safe to request on any UDS node. Text, not scaled live data.
IDENTITY_DIDS: tuple[tuple[int, str], ...] = (
    (0xF190, "vin"),
    (0xF187, "spare_part_number"),
    (0xF18A, "system_supplier"),
    (0xF18C, "ecu_serial"),
)

VIN_DID = IsoDataIdentifier.VIN

# ISO 15765-4 physical addresses (11-bit). Functional 0x7DF is not used — multi-response ISO-TP is messy.
ISO15765_4_MODULES: tuple[Module, ...] = (
    Module("ECM", 0x7E0, 0x7E8, "powertrain", confirmed=True),
    Module("TCM", 0x7E1, 0x7E9, "powertrain", confirmed=False),
    Module("ECU_7E2", 0x7E2, 0x7EA, "powertrain", confirmed=False),
)


@dataclass(frozen=True)
class VehicleProfile:
    id: str
    make: str
    model: str
    year: int
    modules: tuple[Module, ...]
    dids: tuple[DataIdentifier, ...] = ()
    vin_did: int = VIN_DID
    io_controls: tuple[int, ...] = ()
    depth: str = "iso_15765_4"  # captured | toyota_probe | iso_15765_4
    gaps: tuple[str, ...] = ()
    extra_gaps: tuple[str, ...] = field(default=())

    def coverage(self) -> dict[str, object]:
        return {
            "id": self.id,
            "depth": self.depth,
            "gaps": list(self.gaps) + list(self.extra_gaps),
        }

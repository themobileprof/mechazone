"""VIN → UDS profile. Captured maps first; never invent OEM addresses."""

from __future__ import annotations

from mechazone_worker.profiles import avensis_3zr_fae as avensis
from mechazone_worker.profiles import generic_uds, toyota_common
from mechazone_worker.profiles.base import IDENTITY_DIDS, VIN_DID, VehicleProfile

# Public WMI prefixes (ISO 3780). Used only to pick a probe class, not a diagnosis.
TOYOTA_WMI = (
    "JTD", "JTE", "JTN", "JT2", "JT3", "JT4", "JT5", "JT6", "JT8",
    "4T1", "4T3", "4T4", "5TD", "5TE", "5TF",
    "SB1", "AHT", "NMT", "MR0", "MR2", "MM7", "MM8", "PN1", "PN4",
    "2T1", "2T2", "2T3", "3TM", "4TA",
)

TESLA_WMI = ("5YJ", "7SA", "LRW", "XP7")

# Conservative China EV WMIs — not every L* VIN is an EV.
CHINA_EV_WMI = (
    "LGX", "LG8", "LC0", "L6T", "L6P",  # BYD-class
    "LKG", "LK6",  # GAC / Aion-class
    "LB3", "LBE",  # Geely-class
    "LVT", "LVV",  # Chery-class
)


def _wmi(vin: str) -> str:
    return vin.strip().upper()[:3]


def _is_toyota(vin: str, make: str) -> bool:
    if make.strip().lower() in {"toyota", "lexus"}:
        return True
    return _wmi(vin) in TOYOTA_WMI


def _avensis(vin: str, model: str) -> bool:
    if model.strip().lower().startswith("avensis"):
        return True
    v = vin.strip().upper()
    if v == avensis.MOCK_VIN:
        return True
    # Bench / T27 pattern used in this repo — not every SB1KV is proven Avensis.
    return v.startswith("SB1KV")


def extra_gaps_for(vin: str) -> tuple[str, ...]:
    wmi = _wmi(vin)
    if wmi in TESLA_WMI:
        return (
            "Tesla-class VIN: OBD UDS is not a service path. Need a captured map or say so in gaps.",
        )
    if wmi in CHINA_EV_WMI:
        return (
            "China EV WMI: BMS/VCU IDs and CAN-FD/DoIP are not on file. Capture on OpenPort before claiming a diagnosis.",
        )
    return ()


def select_profile(vin: str, make: str = "", model: str = "", year: int = 0) -> VehicleProfile:
    vin = (vin or "").strip().upper()
    make = (make or "").strip()
    model = (model or "").strip()
    extra = extra_gaps_for(vin)
    if vin and _avensis(vin, model):
        return avensis.PROFILE
    if vin and _is_toyota(vin, make):
        p = toyota_common.PROFILE
        return VehicleProfile(
            id=p.id,
            make=make or p.make,
            model=model,
            year=year or p.year,
            modules=p.modules,
            dids=p.dids,
            depth=p.depth,
            gaps=p.gaps,
            extra_gaps=extra,
        )
    p = generic_uds.PROFILE
    return VehicleProfile(
        id=p.id,
        make=make,
        model=model,
        year=year,
        modules=p.modules,
        dids=p.dids,
        depth=p.depth,
        gaps=p.gaps,
        extra_gaps=extra,
    )


__all__ = [
    "IDENTITY_DIDS",
    "VIN_DID",
    "VehicleProfile",
    "avensis",
    "generic_uds",
    "select_profile",
    "toyota_common",
]

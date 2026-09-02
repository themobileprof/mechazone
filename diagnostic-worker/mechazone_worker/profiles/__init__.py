"""VIN → UDS profile. Captured maps first; never invent OEM addresses."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

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


@dataclass(frozen=True)
class CapturedPlatform:
    """One live OpenPort capture. Register another file here — do not special-case in session."""

    profile: VehicleProfile
    matches: Callable[[str, str, str, int], bool]
    mock_replies: Callable[[int], dict[bytes, bytes] | None]
    mock_stream: Callable[[], dict] | None = None


# Platforms we have a captured map for. Avensis is one of them, not the product default.
CAPTURED: tuple[CapturedPlatform, ...] = (
    CapturedPlatform(
        profile=avensis.PROFILE,
        matches=avensis.matches,
        mock_replies=avensis.mock_replies,
        mock_stream=avensis.mock_stream,
    ),
)


def _wmi(vin: str) -> str:
    return vin.strip().upper()[:3]


def _is_toyota(vin: str, make: str) -> bool:
    if vin:
        return _wmi(vin) in TOYOTA_WMI
    return make.strip().lower() in {"toyota", "lexus"}


def _compatible(vin: str, make: str, profile: VehicleProfile) -> bool:
    """A captured map must not attach to a VIN from a different maker."""
    del make
    if not vin:
        return True
    if profile.make.strip().lower() in {"toyota", "lexus"}:
        return _wmi(vin) in TOYOTA_WMI
    return True


def extra_gaps_for(vin: str) -> tuple[str, ...]:
    wmi = _wmi(vin)
    if wmi in TESLA_WMI:
        return (
            "Tesla-class VIN: OBD UDS is not a service path. Do not wrap Tesla Toolbox. "
            "If you have a captured Pass-Thru map, file it; otherwise the playbook stays in gaps.",
        )
    if wmi in CHINA_EV_WMI:
        return (
            "China EV WMI: bring DLC bus type (CAN 500k vs CAN-FD vs DoIP), BMS/VCU tx/rx from a live OpenPort session, "
            "and any licensed manual. This clone is J2534-1 CAN — if the car is CAN-FD/DoIP only, write that down.",
        )
    return ()


def _bind(p: VehicleProfile, make: str, model: str, year: int, extra: tuple[str, ...]) -> VehicleProfile:
    return VehicleProfile(
        id=p.id,
        make=make or p.make,
        model=model or p.model,
        year=year or p.year,
        modules=p.modules,
        dids=p.dids,
        vin_did=p.vin_did,
        io_controls=p.io_controls,
        depth=p.depth,
        gaps=p.gaps,
        extra_gaps=extra,
    )


def captured_by_id(profile_id: str) -> CapturedPlatform | None:
    want = profile_id.strip().lower()
    for cap in CAPTURED:
        if cap.profile.id == want:
            return cap
    return None


def mock_replies_for(profile_id: str | None) -> Callable[[int], dict[bytes, bytes] | None]:
    pid = (profile_id or "").strip().lower()
    if pid in {"", "generic", "generic_uds"}:
        return generic_uds.mock_replies
    cap = captured_by_id(pid)
    if cap is None:
        raise ValueError(f"unknown mock profile {profile_id!r}")
    return cap.mock_replies


def mock_stream_for(profile_id: str) -> Callable[[], dict] | None:
    cap = captured_by_id(profile_id)
    if cap is None:
        return None
    return cap.mock_stream


def select_profile(vin: str, make: str = "", model: str = "", year: int = 0) -> VehicleProfile:
    vin = (vin or "").strip().upper()
    make = (make or "").strip()
    model = (model or "").strip()
    extra = extra_gaps_for(vin)
    for cap in CAPTURED:
        if not cap.matches(vin, make, model, year):
            continue
        if not _compatible(vin, make, cap.profile):
            continue
        return _bind(cap.profile, make, model, year, extra)
    if _is_toyota(vin, make):
        return _bind(toyota_common.PROFILE, make, model, year, extra)
    return _bind(generic_uds.PROFILE, make, model, year, extra)


__all__ = [
    "CAPTURED",
    "IDENTITY_DIDS",
    "VIN_DID",
    "VehicleProfile",
    "captured_by_id",
    "generic_uds",
    "mock_replies_for",
    "mock_stream_for",
    "select_profile",
    "toyota_common",
]

"""USB VID/PID probe for kits we can name. No invented dongle drivers."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path
from typing import Any

from mechazone_worker.j2534 import resolve_j2534_lib

# OpenPort 2.0 Rev E clone — same IDs as deploy/99-openport.rules
OPENPORT_IDS = {("0403", "cc4d"), ("0403", "cca2")}

# Common ELM327 / STN USB-serial chips. Presence ≠ UDS Pass-Thru.
ELM_IDS = {
    ("1a86", "7523"),  # CH340
    ("1a86", "5523"),
    ("1a86", "7522"),
    ("0403", "6001"),  # FTDI — many devices; labeled maybe-ELM
    ("10c4", "ea60"),  # CP210x
    ("067b", "2303"),  # Prolific
    ("0918", "7104"),  # QBD / cheap ELM "Virtual COM Port"
}

# Host internals — never offer these as dongles.
SKIP_VIDS = {
    "1d6b",  # Linux Foundation hubs
    "8087",  # Intel
    "8086",
    "04f2",  # Chicony cameras
    "0bda",  # Realtek wifi/BT
    "13d3",  # Azurewave cameras
}
# USB base class: 09 hub, 0e video, e0 wireless, 03 hid
SKIP_CLASSES = {"09", "0e", "e0", "03"}


def _j2534_lib_path() -> str | None:
    return resolve_j2534_lib()


def _sysfs_class(node: Path) -> str:
    p = node / "bDeviceClass"
    if not p.is_file():
        return ""
    return p.read_text(encoding="ascii", errors="ignore").strip().lower()


def _usb_linux() -> list[tuple[str, str]]:
    root = Path("/sys/bus/usb/devices")
    if not root.is_dir():
        return []
    found: list[tuple[str, str]] = []
    for node in root.iterdir():
        vid_p = node / "idVendor"
        pid_p = node / "idProduct"
        if not vid_p.is_file() or not pid_p.is_file():
            continue
        vid = vid_p.read_text(encoding="ascii", errors="ignore").strip().lower()
        pid = pid_p.read_text(encoding="ascii", errors="ignore").strip().lower()
        if len(vid) != 4 or len(pid) != 4:
            continue
        if vid in SKIP_VIDS:
            continue
        if _sysfs_class(node) in SKIP_CLASSES:
            continue
        found.append((vid, pid))
    return found


def _usb_windows() -> list[tuple[str, str]]:
    try:
        proc = subprocess.run(
            [
                "powershell",
                "-NoProfile",
                "-Command",
                "Get-PnpDevice -PresentOnly -Class USB | ForEach-Object { $_.InstanceId }",
            ],
            capture_output=True,
            text=True,
            timeout=8,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return []
    found: list[tuple[str, str]] = []
    for m in re.finditer(r"VID_([0-9A-Fa-f]{4}).*PID_([0-9A-Fa-f]{4})", proc.stdout):
        found.append((m.group(1).lower(), m.group(2).lower()))
    return found


def usb_ids() -> list[tuple[str, str]]:
    if sys.platform.startswith("linux"):
        return _usb_linux()
    if sys.platform.startswith("win"):
        return _usb_windows()
    return []


def detect_adapters(usb: list[tuple[str, str]] | None = None) -> dict[str, Any]:
    ids = usb if usb is not None else usb_ids()
    seen = set(ids)
    lib = _j2534_lib_path()
    openport_hit = [f"{v}:{p}" for v, p in ids if (v, p) in OPENPORT_IDS]
    elm_hit = [f"{v}:{p}" for v, p in ids if (v, p) in ELM_IDS and (v, p) not in OPENPORT_IDS]

    devices: list[dict[str, Any]] = []
    openport_present = bool(openport_hit)
    devices.append(
        {
            "id": "openport2_rev_e",
            "label": "OpenPort 2.0 Rev E",
            "vid_pid": openport_hit[0] if openport_hit else None,
            "capability": "uds_j2534",
            "present": openport_present,
            "connectable": True,
            "recommended": openport_present,
            "library": lib,
            "gap": None if lib or openport_present else "J2534 library not on this laptop — compile third_party/j2534 or set J2534_LIB.",
        }
    )
    if elm_hit:
        devices.append(
            {
                "id": "elm327",
                "label": "ELM327-class USB serial",
                "vid_pid": elm_hit[0],
                "capability": "detect_only",
                "present": True,
                "connectable": False,
                "recommended": False,
                "library": None,
                "gap": "Seen on USB. UDS Pass-Thru is not wired on ELM — use the OpenPort. python-can / can-isotp are in the venv for a later thin fallback, not J1979.",
            }
        )
    unknown = [f"{v}:{p}" for v, p in ids if (v, p) not in OPENPORT_IDS and (v, p) not in ELM_IDS]
    for vp in unknown[:6]:
        devices.append(
            {
                "id": f"usb_{vp.replace(':', '_')}",
                "label": f"Unknown USB {vp}",
                "vid_pid": vp,
                "capability": "unknown",
                "present": True,
                "connectable": False,
                "recommended": False,
                "library": None,
                "gap": "Unlisted VID:PID. If this is a J2534 Pass-Thru, point J2534_LIB at its library. Do not download a Tactrix official DLL.",
            }
        )
    devices.append(
        {
            "id": "mock",
            "label": "Mock ECU (bench)",
            "vid_pid": None,
            "capability": "uds_mock",
            "present": True,
            "connectable": True,
            "recommended": not openport_present,
            "library": None,
            "gap": None,
        }
    )
    return {
        "devices": devices,
        "j2534_lib": lib,
        "usb": [f"{v}:{p}" for v, p in seen],
        "recommended": "openport2_rev_e" if openport_present else "mock",
    }

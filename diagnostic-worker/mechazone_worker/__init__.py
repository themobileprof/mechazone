"""Mechazone local diagnostic worker — OpenPort Pass-Thru via udsoncan."""

from __future__ import annotations

import ctypes
import sys

# udsoncan.j2534 binds PassThru with WINFUNCTYPE (Windows stdcall).
# Linux j2534.so is cdecl — alias before that module loads.
if sys.platform != "win32" and not hasattr(ctypes, "WINFUNCTYPE"):
    ctypes.WINFUNCTYPE = ctypes.CFUNCTYPE  # type: ignore[misc, assignment]

__version__ = "0.1.0"

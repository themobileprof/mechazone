"""Hex encode/decode for ledger excerpts. Framing itself is udsoncan."""

from __future__ import annotations

from udsoncan.common.dtc import Dtc


def to_hex(data: bytes) -> str:
    return " ".join(f"{b:02X}" for b in data)


def from_hex(text: str) -> bytes:
    cleaned = text.replace(" ", "").replace("0x", "").replace("0X", "")
    if len(cleaned) % 2:
        raise ValueError("odd-length hex")
    return bytes.fromhex(cleaned)


def decode_dtc(high: int, low: int) -> str:
    """ISO 15031-6 via udsoncan.Dtc — not a local encyclopedia."""
    return Dtc((high << 16) | (low << 8)).id_iso().split("-")[0]

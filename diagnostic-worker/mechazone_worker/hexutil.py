from __future__ import annotations


def to_hex(data: bytes) -> str:
    return " ".join(f"{b:02X}" for b in data)


def from_hex(text: str) -> bytes:
    cleaned = text.replace(" ", "").replace("0x", "").replace("0X", "")
    if len(cleaned) % 2:
        raise ValueError("odd-length hex")
    return bytes.fromhex(cleaned)


def decode_dtc(high: int, low: int) -> str:
    """ISO 15031-6 / UDS 2-byte DTC to P/C/B/U string."""
    prefix = ("P", "C", "B", "U")[(high & 0xC0) >> 6]
    first = (high & 0x30) >> 4
    d2 = high & 0x0F
    return f"{prefix}{first}{d2:X}{low:02X}"

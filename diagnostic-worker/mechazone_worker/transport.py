"""UDS connection adapters: J2534 ISO-TP (primary) and an in-process mock."""

from __future__ import annotations

from typing import Protocol

from udsoncan.connections import BaseConnection

from mechazone_worker.hexutil import to_hex
from mechazone_worker.j2534 import Channel, J2534Library


class HexLog(Protocol):
    def record(self, direction: str, payload: bytes) -> None: ...


class MemoryHexLog:
    def __init__(self) -> None:
        self.lines: list[str] = []

    def record(self, direction: str, payload: bytes) -> None:
        self.lines.append(f"{direction} {to_hex(payload)}")


class J2534IsoTpConnection(BaseConnection):
    def __init__(self, lib: J2534Library, channel: Channel, tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> None:
        super().__init__(name=f"j2534:{tx_id:03X}/{rx_id:03X}")
        self._lib = lib
        self._channel = channel
        self._tx_id = tx_id
        self._rx_id = rx_id
        self._hexlog = hexlog
        self._opened = False

    def open(self) -> J2534IsoTpConnection:
        self._lib.start_isotp_filter(self._channel.channel_id, self._tx_id, self._rx_id)
        self._opened = True
        return self

    def close(self) -> None:
        self._opened = False

    def is_open(self) -> bool:
        return self._opened

    def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
        self._hexlog.record("TX", payload)
        timeout_ms = int((timeout or 0.1) * 1000)
        self._lib.write_uds(self._channel.channel_id, self._tx_id, payload, timeout_ms)

    def specific_wait_frame(self, timeout: float | None = None) -> bytes | None:
        timeout_ms = int((timeout or 0.5) * 1000)
        data = self._lib.read_uds(self._channel.channel_id, timeout_ms)
        if data is None:
            return None
        self._hexlog.record("RX", data)
        return data

    def empty_rxqueue(self) -> None:
        return


class MockIsoTpConnection(BaseConnection):
    """Responds to UDS requests from a scripted ECU map. silent=True = dark node (timeout)."""

    def __init__(
        self,
        replies: dict[bytes, bytes] | None,
        hexlog: MemoryHexLog,
        name: str = "mock",
        silent: bool = False,
    ) -> None:
        super().__init__(name=name)
        self._replies = replies or {}
        self._hexlog = hexlog
        self._pending: bytes | None = None
        self._silent = silent
        self.opened = False

    def open(self) -> MockIsoTpConnection:
        self.opened = True
        return self

    def close(self) -> None:
        self.opened = False

    def is_open(self) -> bool:
        return self.opened

    def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
        self._hexlog.record("TX", payload)
        if self._silent:
            self._pending = None
            return
        self._pending = self._replies.get(bytes(payload))
        if self._pending is None:
            sid = payload[0] if payload else 0x00
            self._pending = bytes([0x7F, sid, 0x11])

    def specific_wait_frame(self, timeout: float | None = None) -> bytes | None:
        if self._pending is None:
            return None
        data = self._pending
        self._pending = None
        self._hexlog.record("RX", data)
        return data

    def empty_rxqueue(self) -> None:
        self._pending = None

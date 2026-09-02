"""UDS connections: udsoncan J2534 bindings + FakeConnection. Hex log is ours (ledger)."""

from __future__ import annotations

from typing import Protocol

from udsoncan.connections import BaseConnection, FakeConnection

from mechazone_worker.hexutil import to_hex
from mechazone_worker.j2534 import PassThru


class HexLog(Protocol):
    def record(self, direction: str, payload: bytes) -> None: ...


class MemoryHexLog:
    def __init__(self) -> None:
        self.lines: list[str] = []

    def record(self, direction: str, payload: bytes) -> None:
        self.lines.append(f"{direction} {to_hex(payload)}")


class J2534IsoTpConnection(BaseConnection):
    """Per-module filter on a shared PassThru. close() does not drop the device."""

    def __init__(self, pt: PassThru, tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> None:
        super().__init__(name=f"j2534:{tx_id:03X}/{rx_id:03X}")
        self._pt = pt
        self._tx_id = tx_id
        self._rx_id = rx_id
        self._hexlog = hexlog
        self._opened = False

    def open(self) -> J2534IsoTpConnection:
        self._pt.set_can_ids(self._tx_id, self._rx_id)
        self._opened = True
        return self

    def close(self) -> None:
        self._opened = False

    def is_open(self) -> bool:
        return self._opened

    def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
        self._hexlog.record("TX", payload)
        timeout_ms = int((timeout or 0.1) * 1000)
        self._pt.write_uds(payload, timeout_ms)

    def specific_wait_frame(self, timeout: float | None = None) -> bytes | None:
        timeout_ms = int((timeout or 0.5) * 1000)
        data = self._pt.read_uds(timeout_ms)
        if data is None:
            return None
        self._hexlog.record("RX", data)
        return data

    def empty_rxqueue(self) -> None:
        return


class ScriptedEcu(FakeConnection):
    """udsoncan FakeConnection plus hex log and silent (dark) nodes."""

    def __init__(
        self,
        replies: dict[bytes, bytes],
        hexlog: MemoryHexLog,
        name: str = "mock",
        silent: bool = False,
    ) -> None:
        super().__init__(name=name)
        self.ResponseData = replies
        self._hexlog = hexlog
        self._silent = silent

    def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
        self._hexlog.record("TX", payload)
        if self._silent:
            return
        resp = self.ResponseData.get(bytes(payload))
        if resp is None:
            sid = payload[0] if payload else 0x00
            resp = bytes([0x7F, sid, 0x11])
        self._hexlog.record("RX", resp)
        self.rxqueue.put(resp)

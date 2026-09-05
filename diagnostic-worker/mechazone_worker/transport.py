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
        if not data:
            return None
        self._hexlog.record("RX", data)
        return data

    def empty_rxqueue(self) -> None:
        return


class ScriptedEcu(FakeConnection):
    """udsoncan FakeConnection plus hex log and silent (dark) nodes.

    UDS $14 (ClearDiagnosticInformation) is ISO 14229, not a captured $2F.
    Unless the script pins an NRC, a $14 request clears later $19 lists on this node.
    """

    def __init__(
        self,
        replies: dict[bytes, bytes],
        hexlog: MemoryHexLog,
        name: str = "mock",
        silent: bool = False,
        tx_id: int = 0,
        cleared_tx: set[int] | None = None,
    ) -> None:
        super().__init__(name=name)
        self.ResponseData = replies
        self._hexlog = hexlog
        self._silent = silent
        self._tx_id = tx_id
        self._cleared_tx = cleared_tx
        self._dtcs_cleared = bool(cleared_tx is not None and tx_id in cleared_tx)

    def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
        self._hexlog.record("TX", payload)
        if self._silent:
            return
        key = bytes(payload)
        sid = key[0] if key else 0x00
        if sid == 0x14:
            resp = self.ResponseData.get(key, b"\x54")
            if resp[:1] == b"\x54":
                self._dtcs_cleared = True
                if self._cleared_tx is not None:
                    self._cleared_tx.add(self._tx_id)
        elif self._dtcs_cleared and sid == 0x19:
            rest = key[1:] if len(key) > 1 else b"\x02\xff"
            resp = bytes([0x59]) + rest
        else:
            resp = self.ResponseData.get(key)
            if resp is None:
                resp = bytes([0x7F, sid, 0x11])
        self._hexlog.record("RX", resp)
        self.rxqueue.put(resp)

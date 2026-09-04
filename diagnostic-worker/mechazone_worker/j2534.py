"""Clone-safe Pass-Thru using udsoncan.j2534 — not a second ctypes stack.

Loads only J2534_LIB (absolute path). No Windows J2534 registry. No firmware IOCTLs.
Linux: package __init__ maps WINFUNCTYPE → CFUNCTYPE so udsoncan.j2534 imports.
"""

from __future__ import annotations

import logging
import os
import sys
import time
from pathlib import Path

from udsoncan.j2534 import (
    Error_ID,
    Ioctl_ID,
    J2534,
    Protocol_ID,
    SCONFIG_LIST,
)

log = logging.getLogger("mechazone.passthru")

# SAE J2534-1 RxStatus bits that are not a completed ISO 15765 PDU.
# udsoncan.j2534.PassThruReadMsgs only returns RxStatus in {0, 0x100, 0x110}.
_RX_TX_MSG_TYPE = 0x00000001
_RX_START_OF_MESSAGE = 0x00000002
_RX_TX_INDICATION = 0x00000008
_RX_SKIP = _RX_TX_MSG_TYPE | _RX_START_OF_MESSAGE | _RX_TX_INDICATION


def completed_uds_payload(rx_status: int, data_size: int, data: bytes) -> bytes | None:
    """Strip the 4-byte CAN ID from a PassThru ISO15765 message.

    START_OF_MESSAGE / TX indications often have DataSize 4 (ID only). Treating
    that as a UDS reply makes udsoncan raise 'Payload is empty'.
    """
    if rx_status & _RX_SKIP:
        return None
    if data_size <= 4:
        return None
    payload = data[4:data_size]
    return payload or None


def default_lib_candidates() -> list[str]:
    env = os.environ.get("J2534_LIB", "").strip()
    if env:
        return [env]
    if sys.platform.startswith("win"):
        return []
    root = Path(__file__).resolve().parents[2]
    return [
        str(root / "third_party" / "j2534" / "j2534" / "j2534.so"),
        str(root / "third_party" / "j2534" / "j2534" / "libj2534.so"),
        "/usr/local/lib/j2534.so",
        "/usr/local/lib/libj2534.so",
        "/usr/lib/libopenport.so",
    ]


def resolve_j2534_lib() -> str | None:
    for cand in default_lib_candidates():
        if cand and Path(cand).is_file():
            return cand
    env = os.environ.get("J2534_LIB", "").strip()
    return env or None


def _ok(result: Error_ID) -> bool:
    return result in (Error_ID.ERR_SUCCESS, Error_ID.STATUS_NOERROR)


class CloneSafeJ2534(J2534):
    """Same bindings as udsoncan; PassThruOpen(NULL) so we never name a Tactrix device."""

    def PassThruOpen(self, pDeviceID=None):  # noqa: N802 — SAE name
        import ctypes

        import udsoncan.j2534 as j2534_mod

        if not pDeviceID:
            pDeviceID = ctypes.c_ulong()
        result = j2534_mod.dllPassThruOpen(None, ctypes.byref(pDeviceID))
        return Error_ID(hex(result)), pDeviceID


class PassThru:
    """One open ISO 15765 channel. Filter-switch per module; do not PassThruClose until disconnect."""

    def __init__(self, lib_path: str | None = None) -> None:
        path = lib_path or resolve_j2534_lib()
        if not path:
            raise FileNotFoundError(
                "J2534 library not found. Compile third_party/j2534 or set J2534_LIB to the frozen clone DLL."
            )
        self.path = path
        self.iface = CloneSafeJ2534(windll=path, rxid=0x7E8, txid=0x7E0)
        result, self.dev_id = self.iface.PassThruOpen()
        if not _ok(result):
            raise RuntimeError(f"PassThruOpen failed: {result}")
        result, self.channel_id = self.iface.PassThruConnect(
            self.dev_id, Protocol_ID.ISO15765.value, 500000
        )
        if not _ok(result):
            raise RuntimeError(f"PassThruConnect failed: {result}")
        configs = SCONFIG_LIST(
            [
                (Ioctl_ID.DATA_RATE.value, 500000),
                (Ioctl_ID.LOOPBACK.value, 0),
                (Ioctl_ID.ISO15765_BS.value, 0x20),
                (Ioctl_ID.ISO15765_STMIN.value, 0),
            ]
        )
        self.iface.PassThruIoctl(self.channel_id, Ioctl_ID.SET_CONFIG, configs)
        self.set_can_ids(0x7E0, 0x7E8)

    def set_can_ids(self, tx_id: int, rx_id: int) -> None:
        self.iface.txid = tx_id.to_bytes(4, "big")
        self.iface.rxid = rx_id.to_bytes(4, "big")
        eleven = tx_id <= 0x7FF
        from udsoncan.j2534 import TxStatusFlag

        flags = TxStatusFlag.ISO15765_CAN_ID_11.value if eleven else TxStatusFlag.ISO15765_CAN_ID_29.value
        self.iface.txConnectFlags = flags
        # Enum reuses 0x40 as CAN_ID_11; SAE ISO15765_FRAME_PAD is the same bit.
        self.iface.txFlags = flags | TxStatusFlag.ISO15765_CAN_ID_11.value
        self.iface.PassThruIoctl(self.channel_id, Ioctl_ID.CLEAR_MSG_FILTERS)
        self.iface.PassThruStartMsgFilter(self.channel_id, Protocol_ID.ISO15765.value)
        self.iface.PassThruIoctl(self.channel_id, Ioctl_ID.CLEAR_RX_BUFFER)
        self.iface.PassThruIoctl(self.channel_id, Ioctl_ID.CLEAR_TX_BUFFER)

    def write_uds(self, payload: bytes, timeout_ms: int = 100) -> None:
        result = self.iface.PassThruWriteMsgs(
            self.channel_id, payload, Protocol_ID.ISO15765.value, Timeout=timeout_ms
        )
        if not _ok(result):
            raise RuntimeError(f"PassThruWriteMsgs failed: {result}")

    def read_uds(self, timeout_ms: int = 500) -> bytes | None:
        # Do not use udsoncan.PassThruReadMsgs — it busy-loops on ERR_TIMEOUT.
        # Still skip ISO15765 indications the same way that helper does.
        import ctypes

        import udsoncan.j2534 as j2534_mod

        deadline = time.monotonic() + max(timeout_ms, 1) / 1000.0
        while True:
            remaining_ms = int((deadline - time.monotonic()) * 1000)
            if remaining_ms <= 0:
                return None
            msg = j2534_mod.PASSTHRU_MSG()
            msg.ProtocolID = Protocol_ID.ISO15765.value
            n = ctypes.c_ulong(1)
            result = j2534_mod.dllPassThruReadMsgs(
                self.channel_id, ctypes.byref(msg), ctypes.byref(n), ctypes.c_ulong(remaining_ms)
            )
            err = Error_ID(hex(result))
            if err in (Error_ID.ERR_TIMEOUT, Error_ID.ERR_BUFFER_EMPTY) or n.value == 0:
                return None
            if not _ok(err):
                raise RuntimeError(f"PassThruReadMsgs failed: {err}")
            raw = bytes(msg.Data[: msg.DataSize])
            payload = completed_uds_payload(int(msg.RxStatus), int(msg.DataSize), raw)
            if payload is None:
                log.debug(
                    "skip ISO15765 RxStatus=0x%X DataSize=%s",
                    int(msg.RxStatus),
                    int(msg.DataSize),
                )
                continue
            return payload

    def close(self) -> None:
        try:
            self.iface.PassThruDisconnect(self.channel_id)
        finally:
            self.iface.PassThruClose(self.dev_id)

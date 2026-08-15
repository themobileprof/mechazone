"""Thin ctypes binding for SAE J2534-1 Pass-Thru (OpenPort 2.0 Rev E)."""

from __future__ import annotations

import ctypes
import os
import sys
from ctypes import POINTER, c_char_p, c_ulong, c_void_p, c_ubyte
from dataclasses import dataclass
from typing import Optional

STATUS_NOERROR = 0x00
ERR_TIMEOUT = 0x09
ERR_BUFFER_EMPTY = 0x07

PROTOCOL_CAN = 0x05
PROTOCOL_ISO15765 = 0x06

CAN_29BIT_ID = 0x00000100
ISO15765_FRAME_PAD = 0x00000040
ISO15765_ADDR_TYPE = 0x00000080

SET_CONFIG = 0x01
CLEAR_TX_BUFFER = 0x07
CLEAR_RX_BUFFER = 0x08
CLEAR_MSG_FILTERS = 0x0A
START_MSG_FILTER = 0x0B

PASS_FILTER = 0x00000001
FLOW_CONTROL_FILTER = 0x00000003

DATA_RATE = 0x01


class PASSTHRU_MSG(ctypes.Structure):
    _fields_ = [
        ("ProtocolID", c_ulong),
        ("RxStatus", c_ulong),
        ("TxFlags", c_ulong),
        ("Timestamp", c_ulong),
        ("DataSize", c_ulong),
        ("ExtraDataIndex", c_ulong),
        ("Data", c_ubyte * 4128),
    ]


class SCONFIG(ctypes.Structure):
    _fields_ = [("Parameter", c_ulong), ("Value", c_ulong)]


class SCONFIG_LIST(ctypes.Structure):
    _fields_ = [("NumOfParams", c_ulong), ("ConfigPtr", POINTER(SCONFIG))]


def _default_lib_candidates() -> list[str]:
    env = os.environ.get("J2534_LIB", "").strip()
    if env:
        return [env]
    if sys.platform.startswith("win"):
        return ["openport.dll"]
    return [
        "libopenport.so",
        "/usr/lib/libopenport.so",
        "/usr/local/lib/libopenport.so",
        "/usr/lib/j2534/libopenport.so",
    ]


@dataclass
class Channel:
    device_id: int
    channel_id: int
    protocol: int


class J2534Error(RuntimeError):
    def __init__(self, code: int, action: str) -> None:
        super().__init__(f"J2534 {action} failed with status 0x{code:02X}")
        self.code = code


class J2534Library:
    def __init__(self, path: Optional[str] = None) -> None:
        last_err: Exception | None = None
        candidates = [path] if path else _default_lib_candidates()
        self._lib = None
        self.path = ""
        for cand in candidates:
            if not cand:
                continue
            try:
                self._lib = ctypes.CDLL(cand)
                self.path = cand
                break
            except OSError as exc:
                last_err = exc
        if self._lib is None:
            raise FileNotFoundError(
                f"OpenPort J2534 library not found ({candidates!r}): {last_err}"
            )
        self._bind()

    def _bind(self) -> None:
        lib = self._lib
        lib.PassThruOpen.argtypes = [c_void_p, POINTER(c_ulong)]
        lib.PassThruOpen.restype = c_ulong
        lib.PassThruClose.argtypes = [c_ulong]
        lib.PassThruClose.restype = c_ulong
        lib.PassThruConnect.argtypes = [
            c_ulong, c_ulong, c_ulong, c_ulong, POINTER(c_ulong)
        ]
        lib.PassThruConnect.restype = c_ulong
        lib.PassThruDisconnect.argtypes = [c_ulong]
        lib.PassThruDisconnect.restype = c_ulong
        lib.PassThruReadMsgs.argtypes = [
            c_ulong, POINTER(PASSTHRU_MSG), POINTER(c_ulong), c_ulong
        ]
        lib.PassThruReadMsgs.restype = c_ulong
        lib.PassThruWriteMsgs.argtypes = [
            c_ulong, POINTER(PASSTHRU_MSG), POINTER(c_ulong), c_ulong
        ]
        lib.PassThruWriteMsgs.restype = c_ulong
        lib.PassThruStartMsgFilter.argtypes = [
            c_ulong, c_ulong, POINTER(PASSTHRU_MSG), POINTER(PASSTHRU_MSG),
            POINTER(PASSTHRU_MSG), POINTER(c_ulong),
        ]
        lib.PassThruStartMsgFilter.restype = c_ulong
        lib.PassThruIoctl.argtypes = [c_ulong, c_ulong, c_void_p, c_void_p]
        lib.PassThruIoctl.restype = c_ulong
        lib.PassThruGetLastError.argtypes = [c_char_p]
        lib.PassThruGetLastError.restype = c_ulong

    def last_error(self) -> str:
        buf = ctypes.create_string_buffer(256)
        self._lib.PassThruGetLastError(buf)
        return buf.value.decode("ascii", errors="replace")

    def _check(self, code: int, action: str) -> None:
        if code != STATUS_NOERROR:
            raise J2534Error(code, f"{action}: {self.last_error()}")

    def open(self) -> int:
        device_id = c_ulong(0)
        self._check(self._lib.PassThruOpen(None, ctypes.byref(device_id)), "PassThruOpen")
        return int(device_id.value)

    def close(self, device_id: int) -> None:
        self._check(self._lib.PassThruClose(device_id), "PassThruClose")

    def connect_iso15765(self, device_id: int, baud: int = 500000) -> int:
        channel_id = c_ulong(0)
        flags = ISO15765_FRAME_PAD
        self._check(
            self._lib.PassThruConnect(
                device_id, PROTOCOL_ISO15765, flags, baud, ctypes.byref(channel_id)
            ),
            "PassThruConnect(ISO15765)",
        )
        cfg = SCONFIG(DATA_RATE, baud)
        cfg_list = SCONFIG_LIST(1, ctypes.pointer(cfg))
        self._lib.PassThruIoctl(channel_id.value, SET_CONFIG, ctypes.byref(cfg_list), None)
        self._lib.PassThruIoctl(channel_id.value, CLEAR_TX_BUFFER, None, None)
        self._lib.PassThruIoctl(channel_id.value, CLEAR_RX_BUFFER, None, None)
        self._lib.PassThruIoctl(channel_id.value, CLEAR_MSG_FILTERS, None, None)
        return int(channel_id.value)

    def start_isotp_filter(self, channel_id: int, tx_id: int, rx_id: int) -> int:
        mask = PASSTHRU_MSG()
        pattern = PASSTHRU_MSG()
        flow = PASSTHRU_MSG()
        for msg in (mask, pattern, flow):
            msg.ProtocolID = PROTOCOL_ISO15765
            msg.TxFlags = ISO15765_FRAME_PAD
            msg.DataSize = 4
        _put_can_id(mask, 0xFFFFFFFF)
        _put_can_id(pattern, rx_id)
        _put_can_id(flow, tx_id)
        filter_id = c_ulong(0)
        self._check(
            self._lib.PassThruStartMsgFilter(
                channel_id,
                FLOW_CONTROL_FILTER,
                ctypes.byref(mask),
                ctypes.byref(pattern),
                ctypes.byref(flow),
                ctypes.byref(filter_id),
            ),
            "PassThruStartMsgFilter",
        )
        return int(filter_id.value)

    def write_uds(self, channel_id: int, tx_id: int, payload: bytes, timeout_ms: int = 100) -> None:
        msg = PASSTHRU_MSG()
        msg.ProtocolID = PROTOCOL_ISO15765
        msg.TxFlags = ISO15765_FRAME_PAD
        frame = _can_id_bytes(tx_id) + payload
        msg.DataSize = len(frame)
        for i, b in enumerate(frame):
            msg.Data[i] = b
        count = c_ulong(1)
        self._check(
            self._lib.PassThruWriteMsgs(
                channel_id, ctypes.byref(msg), ctypes.byref(count), timeout_ms
            ),
            "PassThruWriteMsgs",
        )

    def read_uds(self, channel_id: int, timeout_ms: int = 500) -> bytes | None:
        msg = PASSTHRU_MSG()
        msg.ProtocolID = PROTOCOL_ISO15765
        count = c_ulong(1)
        code = self._lib.PassThruReadMsgs(
            channel_id, ctypes.byref(msg), ctypes.byref(count), timeout_ms
        )
        if code in (ERR_TIMEOUT, ERR_BUFFER_EMPTY):
            return None
        if code != STATUS_NOERROR:
            raise J2534Error(code, f"PassThruReadMsgs: {self.last_error()}")
        if count.value == 0 or msg.DataSize < 4:
            return None
        return bytes(msg.Data[4:msg.DataSize])

    def disconnect(self, channel_id: int) -> None:
        self._check(self._lib.PassThruDisconnect(channel_id), "PassThruDisconnect")


def _can_id_bytes(can_id: int) -> bytes:
    return can_id.to_bytes(4, "big")


def _put_can_id(msg: PASSTHRU_MSG, can_id: int) -> None:
    raw = _can_id_bytes(can_id)
    for i, b in enumerate(raw):
        msg.Data[i] = b

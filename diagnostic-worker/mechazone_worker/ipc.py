from __future__ import annotations

import asyncio
import json
import logging
import os
from typing import Any

from mechazone_worker.detect import detect_adapters
from mechazone_worker.j2534 import J2534Library
from mechazone_worker.session import DiagnosticSession, mock_factory, openport_factory

log = logging.getLogger("mechazone.worker")


class WorkerState:
    def __init__(self) -> None:
        self.adapter = os.environ.get("MECHAZONE_ADAPTER", "mock")
        self.session: DiagnosticSession | None = None
        self.lib: J2534Library | None = None
        self.device_id: int | None = None
        self.channel_id: int | None = None

    def connect(self, adapter: str) -> dict[str, Any]:
        self.disconnect()
        self.adapter = adapter
        if adapter == "mock":
            self.session = DiagnosticSession(mock_factory(), "mock")
            return {"adapter": "mock", "library": None, "connected": True}
        if adapter == "elm327":
            raise ValueError(
                "ELM327 is on USB but UDS Pass-Thru is not wired on that cable. Connect the OpenPort 2.0 Rev E."
            )
        if adapter != "openport2_rev_e":
            raise ValueError(f"unknown adapter {adapter}")
        lib = J2534Library()
        device_id = lib.open()
        channel_id = lib.connect_iso15765(device_id)
        from mechazone_worker.j2534 import Channel

        channel = Channel(device_id, channel_id, protocol=0x06)
        self.lib = lib
        self.device_id = device_id
        self.channel_id = channel_id
        self.session = DiagnosticSession(openport_factory(lib, channel), "openport2_rev_e")
        return {"adapter": "openport2_rev_e", "library": lib.path, "connected": True}

    def disconnect(self) -> None:
        if self.lib and self.channel_id is not None:
            try:
                self.lib.disconnect(self.channel_id)
            except Exception as exc:  # noqa: BLE001
                log.warning("disconnect channel: %s", exc)
        if self.lib and self.device_id is not None:
            try:
                self.lib.close(self.device_id)
            except Exception as exc:  # noqa: BLE001
                log.warning("close device: %s", exc)
        self.lib = None
        self.device_id = None
        self.channel_id = None
        self.session = None

    def require(self) -> DiagnosticSession:
        if self.session is None:
            raise RuntimeError("adapter not connected")
        return self.session


async def handle(ws, state: WorkerState) -> None:
    async for raw in ws:
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            await ws.send(json.dumps({"ok": False, "error": "invalid json"}))
            continue
        req_id = msg.get("id")
        cmd = msg.get("cmd")
        try:
            result = await asyncio.to_thread(_dispatch, state, cmd, msg)
            await ws.send(json.dumps({"id": req_id, "ok": True, "result": result}))
        except Exception as exc:  # noqa: BLE001
            log.exception("cmd %s", cmd)
            await ws.send(json.dumps({"id": req_id, "ok": False, "error": str(exc)}))


def _dispatch(state: WorkerState, cmd: str, msg: dict[str, Any]) -> Any:
    if cmd == "status":
        return {
            "connected": state.session is not None,
            "adapter": state.adapter if state.session else None,
        }
    if cmd == "detect":
        return detect_adapters()
    if cmd == "connect":
        adapter = msg.get("adapter") or state.adapter
        return state.connect(adapter)
    if cmd == "disconnect":
        state.disconnect()
        return {"connected": False}
    if cmd == "identify":
        ident = state.require().identify()
        ident["raw_hex"] = state.require().hexlog.lines
        return ident
    if cmd == "scan":
        result = state.require().scan(
            make=str(msg.get("make") or ""),
            model=str(msg.get("model") or ""),
            year=int(msg.get("year") or 0),
        )
        return {
            "vin": result.vin,
            "profile": result.profile,
            "make": result.make,
            "model": result.model,
            "year": result.year,
            "adapter_type": result.adapter_type,
            "protocol": result.protocol,
            "active_codes": result.active_codes,
            "live": result.live,
            "freeze_frame": result.freeze_frame,
            "raw_hex_stream": result.raw_hex_stream,
            "modules": result.modules,
            "circuit_classes": result.circuit_classes,
            "network": result.network,
            "coverage": result.coverage,
            "identity": result.identity,
        }
    if cmd == "stream_dids":
        seconds = float(msg.get("seconds") or 6)
        return state.require().stream_dids(seconds)
    raise ValueError(f"unknown cmd {cmd}")


async def serve(host: str, port: int) -> None:
    from websockets.asyncio.server import serve

    state = WorkerState()
    log.info("worker listening ws://%s:%s adapter=%s", host, port, state.adapter)

    async with serve(lambda ws: handle(ws, state), host, port):
        await asyncio.Future()

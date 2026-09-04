"""WebSocket JSON bridge between the bay UI and DiagnosticSession. No PII in payloads."""

from __future__ import annotations

import asyncio
import json
import logging
import os
from typing import Any

from mechazone_worker.detect import detect_adapters
from mechazone_worker.j2534 import PassThru
from mechazone_worker.session import DiagnosticSession, mock_factory, openport_factory

from websockets.exceptions import ConnectionClosed

log = logging.getLogger("mechazone.worker")


class WorkerState:
    def __init__(self) -> None:
        self.adapter = os.environ.get("MECHAZONE_ADAPTER", "mock")
        self.session: DiagnosticSession | None = None
        self.passthru: PassThru | None = None

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
        pt = PassThru()
        self.passthru = pt
        self.session = DiagnosticSession(openport_factory(pt), "openport2_rev_e")
        return {"adapter": "openport2_rev_e", "library": pt.path, "connected": True}

    def disconnect(self) -> None:
        if self.passthru is not None:
            try:
                self.passthru.close()
            except Exception as exc:  # noqa: BLE001
                log.warning("passthru close: %s", exc)
        self.passthru = None
        self.session = None

    def require(self) -> DiagnosticSession:
        if self.session is None:
            raise RuntimeError("adapter not connected")
        return self.session


async def handle(ws, state: WorkerState) -> None:
    try:
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
    except ConnectionClosed:
        log.info("bay disconnected")


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
        ident = state.require().identify(vin=str(msg.get("vin") or ""))
        ident["raw_hex"] = state.require().hexlog.lines
        return ident
    if cmd == "scan":
        result = state.require().scan(
            make=str(msg.get("make") or ""),
            model=str(msg.get("model") or ""),
            year=int(msg.get("year") or 0),
            vin=str(msg.get("vin") or ""),
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

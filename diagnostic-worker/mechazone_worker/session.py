"""Live UDS session: identify VIN, scan modules, stream DIDs via udsoncan.Client.

One PassThru ISO 15765 channel; do not PassThru-close between modules.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import Any, Callable

from udsoncan.client import Client
from udsoncan.common.DidCodec import DidCodec
from udsoncan.configs import default_client_config
from udsoncan.exceptions import (
    InvalidResponseException,
    NegativeResponseException,
    TimeoutException,
    UnexpectedResponseException,
)

_UDS_MISS = (
    TimeoutException,
    TimeoutError,
    NegativeResponseException,
    InvalidResponseException,
    UnexpectedResponseException,
)

from mechazone_worker.circuit import classify_codes, network_hint
from mechazone_worker.profiles import (
    IDENTITY_DIDS,
    VIN_DID,
    VehicleProfile,
    mock_replies_for,
    mock_stream_for,
    select_profile,
)
from mechazone_worker.profiles.base import ISO15765_4_MODULES
from mechazone_worker.transport import (
    J2534IsoTpConnection,
    MemoryHexLog,
    ScriptedEcu,
)


class RemainingCodec(DidCodec):
    """udsoncan DidCodec with ReadAllRemainingData — VIN and identity strings are not fixed-width."""
    def decode(self, did_payload: bytes) -> bytes:
        return did_payload

    def encode(self, *did_value: object) -> bytes:
        value = did_value[0]
        if isinstance(value, bytes):
            return value
        raise TypeError("RemainingCodec encodes bytes only")

    def __len__(self) -> int:
        raise DidCodec.ReadAllRemainingData


@dataclass
class ScanResult:
    vin: str
    profile: str
    make: str
    model: str
    year: int
    adapter_type: str
    protocol: str
    active_codes: list[str]
    live: list[dict[str, Any]]
    freeze_frame: dict[str, Any]
    raw_hex_stream: list[str]
    modules: list[dict[str, Any]] = field(default_factory=list)
    circuit_classes: list[dict[str, str]] = field(default_factory=list)
    network: dict[str, Any] = field(default_factory=dict)
    coverage: dict[str, Any] = field(default_factory=dict)
    identity: list[dict[str, str]] = field(default_factory=list)


class DiagnosticSession:
    def __init__(
        self,
        connection_factory: Callable[[int, int, MemoryHexLog], Any],
        adapter_type: str,
    ) -> None:
        self._factory = connection_factory
        self.adapter_type = adapter_type
        self.hexlog = MemoryHexLog()
        self.vin = ""
        self.profile: VehicleProfile = select_profile("")

    def identify(self) -> dict[str, Any]:
        vin = ""
        for i, module in enumerate(ISO15765_4_MODULES):
            attempts = 3 if i == 0 and self.adapter_type != "mock" else 1
            for _ in range(attempts):
                try:
                    vin = self._vin_on(module.tx_id, module.rx_id)
                except _UDS_MISS:
                    continue
                if vin:
                    break
            if vin:
                break
        if vin:
            self.vin = vin
            self.profile = select_profile(vin)
        coverage = self.profile.coverage()
        if not vin:
            coverage = dict(coverage)
            coverage["gaps"] = [
                "VIN DID F190 did not answer. Type the VIN or deep-scan anyway — a timeout is a dark node.",
                *list(coverage.get("gaps") or []),
            ]
        return {
            "vin": vin,
            "profile": self.profile.id,
            "make": self.profile.make,
            "model": self.profile.model,
            "year": self.profile.year,
            "coverage": coverage,
        }

    def scan(self, make: str = "", model: str = "", year: int = 0, vin: str = "") -> ScanResult:
        typed = (vin or "").strip().upper()
        if typed:
            self.vin = typed
        self.hexlog = MemoryHexLog()
        # Lock the map from VIN + decode hints before probing. Captured platforms
        # apply when we know the car — they are not the session default.
        self.profile = select_profile(self.vin, make, model, year)
        vin, codes, live, identity, modules = self._probe_profile()

        if vin:
            upgraded = select_profile(vin, make, model, year)
            if upgraded.id != self.profile.id:
                self.profile = upgraded
                vin, codes, live, identity, modules = self._probe_profile()
            else:
                self.profile = upgraded

        make_out = make or self.profile.make
        model_out = model or self.profile.model
        year_out = year or self.profile.year

        freeze = {item["name"]: item["value"] for item in live}
        classes = classify_codes(codes)
        return ScanResult(
            vin=vin,
            profile=self.profile.id,
            make=make_out,
            model=model_out,
            year=year_out,
            adapter_type=self.adapter_type,
            protocol="uds_isotp_can",
            active_codes=codes,
            live=live,
            freeze_frame=freeze,
            raw_hex_stream=list(self.hexlog.lines),
            modules=modules,
            circuit_classes=classes,
            network=network_hint(modules),
            coverage=self.profile.coverage(),
            identity=identity,
        )

    def stream_dids(self, seconds: float = 6.0) -> dict[str, Any]:
        """Sample ECM DIDs for a wiggle test. No IO-control IDs unless captured on the profile."""
        if seconds <= 0:
            seconds = 6.0
        if seconds > 20:
            seconds = 20.0
        if self.adapter_type == "mock":
            canned = mock_stream_for(self.profile.id)
            if canned is not None:
                return canned()
        if not self.profile.dids:
            return {
                "seconds": 0,
                "module": "ECM",
                "tx_id": "0x7E0",
                "io_control": "none_captured",
                "samples": [],
                "raw_hex_stream": [],
                "gap": "No captured numeric DIDs on this profile — wiggle log needs a live OpenPort capture.",
            }
        self.hexlog = MemoryHexLog()
        samples: list[dict[str, Any]] = []
        import time

        deadline = time.monotonic() + seconds
        ecm = next((m for m in self.profile.modules if m.name == "ECM"), ISO15765_4_MODULES[0])
        conn = self._factory(ecm.tx_id, ecm.rx_id, self.hexlog)
        with Client(conn, request_timeout=1.0, config=_uds_config(self.profile, 1.0)) as client:
            while time.monotonic() < deadline:
                row: dict[str, Any] = {"t": round(seconds - (deadline - time.monotonic()), 2), "values": {}}
                for item in self.profile.dids:
                    try:
                        resp = client.read_data_by_identifier(item.did)
                        raw = resp.service_data.values[item.did]
                        if isinstance(raw, (bytes, bytearray)):
                            row["values"][item.name] = _decode_scaled(bytes(raw), item.size, item.scale, item.offset)
                    except _UDS_MISS:
                        row["values"][item.name] = None
                samples.append(row)
                time.sleep(0.35)
        return {
            "seconds": seconds,
            "module": "ECM",
            "tx_id": f"0x{ecm.tx_id:03X}",
            "io_control": "none_captured",
            "samples": samples,
            "raw_hex_stream": list(self.hexlog.lines),
        }

    def _probe_profile(self) -> tuple[str, list[str], list[dict[str, Any]], list[dict[str, str]], list[dict[str, Any]]]:
        vin = self.vin
        codes: list[str] = []
        live: list[dict[str, Any]] = []
        identity: list[dict[str, str]] = []
        modules: list[dict[str, Any]] = []
        for module in self.profile.modules:
            timeout = 2.0 if module.name == "ECM" else 0.35
            info = self._probe_module(module, timeout)
            if module.name == "ECM" and info.get("vin"):
                vin = str(info["vin"])
                self.vin = vin
            codes.extend(info.get("dtcs") or [])
            live.extend(info.get("live") or [])
            identity.extend(info.get("identity") or [])
            modules.append({k: v for k, v in info.items() if k not in {"live", "identity", "vin"}})
        return vin, codes, live, identity, modules

    def _vin_on(self, tx_id: int, rx_id: int) -> str:
        conn = self._factory(tx_id, rx_id, self.hexlog)
        with Client(conn, request_timeout=2.0, config=_uds_config(self.profile, 2.0)) as client:
            return _read_vin(client, self.profile.vin_did)

    def _probe_module(self, module: Any, timeout: float) -> dict[str, Any]:
        conn = self._factory(module.tx_id, module.rx_id, self.hexlog)
        info: dict[str, Any] = {
            "name": module.name,
            "tx_id": f"0x{module.tx_id:03X}",
            "rx_id": f"0x{module.rx_id:03X}",
            "family": module.family,
            "confirmed": module.confirmed,
            "reachable": False,
            "dtcs": [],
        }
        try:
            with Client(conn, request_timeout=timeout, config=_uds_config(self.profile, timeout)) as client:
                node_codes = _read_dtcs(client)
                info["dtcs"] = node_codes
                info["reachable"] = True
                if module.name == "ECM":
                    try:
                        info["vin"] = _read_vin(client, self.profile.vin_did)
                    except _UDS_MISS:
                        info["vin"] = ""
                    info["live"] = _read_live(client, self.profile)
                    info["identity"] = _read_identity(client)
        except TimeoutException:
            info["error"] = "timeout"
        except TimeoutError:
            info["error"] = "timeout"
        except InvalidResponseException:
            info["error"] = "empty_pdu"
        except UnexpectedResponseException:
            info["error"] = "unexpected"
        except NegativeResponseException:
            info["reachable"] = True
            info["error"] = "nrc"
        return info


def _uds_config(profile: VehicleProfile, timeout: float = 2.0) -> dict[str, Any]:
    cfg = dict(default_client_config)
    dids = {VIN_DID: RemainingCodec(), **{did: RemainingCodec() for did, _ in IDENTITY_DIDS}}
    dids.update({item.did: RemainingCodec() for item in profile.dids})
    cfg["data_identifiers"] = dids
    cfg["p2_timeout"] = timeout
    cfg["request_timeout"] = timeout + 1.0
    return cfg


def _read_vin(client: Client, vin_did: int = VIN_DID) -> str:
    resp = client.read_data_by_identifier(vin_did)
    raw = resp.service_data.values[vin_did]
    if isinstance(raw, bytes):
        return raw.decode("ascii", errors="replace").strip("\x00 ").strip()
    return str(raw)


def _read_dtcs(client: Client) -> list[str]:
    resp = client.get_dtc_by_status_mask(0xFF)
    if resp is None:
        return []
    out: list[str] = []
    for dtc in resp.service_data.dtcs:
        out.append(dtc.id_iso().split("-")[0])
    return out


def _read_live(client: Client, profile: VehicleProfile) -> list[dict[str, Any]]:
    live: list[dict[str, Any]] = []
    for item in profile.dids:
        try:
            resp = client.read_data_by_identifier(item.did)
            raw = resp.service_data.values[item.did]
            if not isinstance(raw, (bytes, bytearray)):
                continue
            value = _decode_scaled(bytes(raw), item.size, item.scale, item.offset)
            live.append(
                {
                    "name": item.name,
                    "value": value,
                    "unit": item.unit,
                    "did": f"0x{item.did:04X}",
                }
            )
        except _UDS_MISS:
            continue
    return live


def _read_identity(client: Client) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for did, name in IDENTITY_DIDS:
        if did == VIN_DID:
            continue
        try:
            resp = client.read_data_by_identifier(did)
            raw = resp.service_data.values[did]
            if isinstance(raw, (bytes, bytearray)):
                text = bytes(raw).decode("ascii", errors="replace").strip("\x00 ").strip()
            else:
                text = str(raw)
            if text:
                rows.append({"name": name, "did": f"0x{did:04X}", "text": text})
        except _UDS_MISS:
            continue
    return rows


def _decode_scaled(raw: bytes, size: int, scale: float, offset: float) -> float:
    chunk = raw[:size] if size else raw
    n = int.from_bytes(chunk, "big")
    return round(n * scale + offset, 3)


def openport_factory(pt):
    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> J2534IsoTpConnection:
        return J2534IsoTpConnection(pt, tx_id, rx_id, hexlog)

    return factory


def mock_factory(profile_id: str | None = None):
    """Bench ECU via udsoncan FakeConnection. Pin a captured map with MECHAZONE_MOCK_PROFILE."""
    pid = profile_id if profile_id is not None else os.environ.get("MECHAZONE_MOCK_PROFILE", "")
    replies_fn = mock_replies_for(pid)

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        replies = replies_fn(tx_id)
        silent = replies is None
        return ScriptedEcu(
            replies or {},
            hexlog,
            name=f"mock:{tx_id:03X}",
            silent=silent,
        )

    return factory


def generic_mock_factory():
    return mock_factory("generic_uds")

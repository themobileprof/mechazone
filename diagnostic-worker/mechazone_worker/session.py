"""Live UDS session: identify VIN, scan modules, stream DIDs, clear DTCs via udsoncan.Client.

One PassThru ISO 15765 channel; do not PassThru-close between modules.
"""

from __future__ import annotations

import os
import time
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
from udsoncan.services import DiagnosticSessionControl

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
        self._last_modules: list[dict[str, Any]] = []

    def identify(self, vin: str = "") -> dict[str, Any]:
        typed = (vin or "").strip().upper()
        bus_vin = ""
        for i, module in enumerate(ISO15765_4_MODULES):
            attempts = 3 if i == 0 and self.adapter_type != "mock" else 1
            for _ in range(attempts):
                try:
                    bus_vin = self._vin_on(module.tx_id, module.rx_id)
                except _UDS_MISS:
                    continue
                if bus_vin:
                    break
            if bus_vin:
                break
        if bus_vin:
            self.vin = bus_vin
            self.profile = select_profile(bus_vin)
        elif typed:
            # Typed VIN is not a bus read — use it only to pick the probe class.
            self.vin = typed
            self.profile = select_profile(typed)
        coverage = self.profile.coverage()
        if not bus_vin:
            coverage = dict(coverage)
            coverage["gaps"] = [
                "Kit VIN (DID F190) stayed dark — normal on many cars. "
                "Type the 17 characters on Vehicle, then deep-scan. Modules still answer.",
                *list(coverage.get("gaps") or []),
            ]
        return {
            "vin": bus_vin,
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
        self._last_modules = modules
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

    def clear_dtcs(self) -> dict[str, Any]:
        """UDS $14 ClearDiagnosticInformation (group 0xFFFFFF) on reachable nodes that have codes.

        Dark modules are not addressed. No security access, no session change, no $2F.
        """
        self.hexlog = MemoryHexLog()
        nodes: list[dict[str, Any]] = []
        before: list[str] = []
        after: list[str] = []
        gaps: list[str] = []
        for module in self.profile.modules:
            prev = next((m for m in self._last_modules if m.get("name") == module.name), None)
            if prev is not None and not prev.get("reachable"):
                nodes.append(
                    {
                        "name": module.name,
                        "tx_id": f"0x{module.tx_id:03X}",
                        "rx_id": f"0x{module.rx_id:03X}",
                        "reachable": False,
                        "attempted": False,
                        "cleared": False,
                        "codes_before": [],
                        "codes_after": [],
                        "error": "timeout",
                    }
                )
                continue
            if prev is not None and not (prev.get("dtcs") or []):
                nodes.append(
                    {
                        "name": module.name,
                        "tx_id": f"0x{module.tx_id:03X}",
                        "rx_id": f"0x{module.rx_id:03X}",
                        "reachable": True,
                        "attempted": False,
                        "cleared": False,
                        "codes_before": [],
                        "codes_after": [],
                    }
                )
                continue
            timeout = 5.0 if module.name == "ECM" else 2.0
            info = self._clear_module(module, timeout)
            nodes.append(info)
            before.extend(info.get("codes_before") or [])
            after.extend(info.get("codes_after") or [])
            if info.get("gap"):
                gaps.append(str(info["gap"]))
            if prev is not None:
                prev["dtcs"] = list(info.get("codes_after") or [])
                prev["reachable"] = bool(info.get("reachable") or prev.get("reachable"))
        before_u = _unique(before)
        after_u = _unique(after)
        if not any(n.get("attempted") for n in nodes):
            raise RuntimeError(
                "No reachable module currently has codes. Deep-scan first. "
                "Dark nodes are not sent UDS $14."
            )
        return {
            "service": "0x14",
            "group": "0xFFFFFF",
            "codes_before": before_u,
            "codes_after": after_u,
            "modules": nodes,
            "circuit_classes": classify_codes(after_u),
            "raw_hex_stream": list(self.hexlog.lines),
            "gaps": gaps,
        }

    def _clear_module(self, module: Any, timeout: float) -> dict[str, Any]:
        info: dict[str, Any] = {
            "name": module.name,
            "tx_id": f"0x{module.tx_id:03X}",
            "rx_id": f"0x{module.rx_id:03X}",
            "reachable": False,
            "attempted": False,
            "cleared": False,
            "codes_before": [],
            "codes_after": [],
        }
        conn = self._factory(module.tx_id, module.rx_id, self.hexlog)
        try:
            with Client(conn, request_timeout=timeout, config=_uds_config(self.profile, timeout)) as client:
                try:
                    codes = _read_dtcs(client)
                except (TimeoutException, TimeoutError):
                    info["error"] = "timeout"
                    return info
                except NegativeResponseException as exc:
                    info["reachable"] = True
                    info["error"] = _nrc_label(exc)
                    info["gap"] = (
                        f"{module.name} answered but $19 {info['error']} — $14 not sent (no security access)."
                    )
                    return info
                except (InvalidResponseException, UnexpectedResponseException):
                    info["error"] = "unexpected"
                    return info
                info["reachable"] = True
                info["codes_before"] = codes
                if not codes:
                    return info
                info["attempted"] = True
                try:
                    group, sess = _clear_dtc(client)
                except NegativeResponseException as exc:
                    label = _nrc_label(exc)
                    info["error"] = label
                    info["codes_after"] = codes
                    info["gap"] = (
                        f"{module.name} rejected $14 ({label}). "
                        "No seed/key retry. Codes left as read."
                    )
                    return info
                info["cleared"] = True
                info["group"] = group
                info["session"] = sess
                time.sleep(0.15)
                try:
                    info["codes_after"] = _read_dtcs(client)
                except _UDS_MISS:
                    info["codes_after"] = []
                    info["gap"] = f"{module.name} accepted $14; re-read DTCs missed."
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


def _unique(codes: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for code in codes:
        if code in seen:
            continue
        seen.add(code)
        out.append(code)
    return out


def _nrc_label(exc: NegativeResponseException) -> str:
    code = _nrc_int(exc)
    if code is not None:
        return f"nrc_0x{code:02X}"
    return "nrc"


def _nrc_int(exc: NegativeResponseException) -> int | None:
    resp = getattr(exc, "response", None)
    code = getattr(resp, "code", None)
    return code if isinstance(code, int) else None


_SESSION_NRC = {0x22, 0x7F}  # conditionsNotCorrect, serviceNotSupportedInActiveSession
_RANGE_NRC = {0x31}  # requestOutOfRange


def _clear_dtc(client: Client) -> tuple[str, str]:
    """UDS $14 group 0xFFFFFF, then 0x000000. Extended session (0x10 03) only — never programming."""
    last: NegativeResponseException | None = None
    for group in (0xFFFFFF, 0x000000):
        try:
            client.clear_dtc(group)
            return f"0x{group:06X}", "default"
        except NegativeResponseException as exc:
            last = exc
            nrc = _nrc_int(exc)
            if nrc in _SESSION_NRC:
                try:
                    client.change_session(DiagnosticSessionControl.Session.extendedDiagnosticSession)
                except NegativeResponseException as sess_exc:
                    raise last from sess_exc
                except (InvalidResponseException, UnexpectedResponseException):
                    pass
                try:
                    client.clear_dtc(group)
                    return f"0x{group:06X}", "extended"
                except NegativeResponseException as exc2:
                    last = exc2
                    nrc = _nrc_int(exc2)
                    if nrc in _RANGE_NRC:
                        continue
                    raise
            elif nrc in _RANGE_NRC:
                continue
            else:
                raise
    if last is not None:
        raise last
    raise RuntimeError("UDS $14 returned no response")


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
    cleared_tx: set[int] = set()

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del rx_id
        replies = replies_fn(tx_id)
        silent = replies is None
        return ScriptedEcu(
            replies or {},
            hexlog,
            name=f"mock:{tx_id:03X}",
            silent=silent,
            tx_id=tx_id,
            cleared_tx=cleared_tx,
        )

    return factory


def generic_mock_factory():
    return mock_factory("generic_uds")

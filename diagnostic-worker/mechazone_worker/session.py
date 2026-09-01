from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Callable

from udsoncan.client import Client
from udsoncan.common.DidCodec import DidCodec
from udsoncan.configs import default_client_config
from udsoncan.exceptions import NegativeResponseException, TimeoutException

from mechazone_worker.circuit import classify_codes, network_hint
from mechazone_worker.hexutil import decode_dtc
from mechazone_worker.profiles import avensis_3zr_fae as avensis
from mechazone_worker.transport import (
    J2534IsoTpConnection,
    MemoryHexLog,
    MockIsoTpConnection,
)


class RemainingCodec(DidCodec):
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


class DiagnosticSession:
    def __init__(
        self,
        connection_factory: Callable[[int, int, MemoryHexLog], Any],
        adapter_type: str,
    ) -> None:
        self._factory = connection_factory
        self.adapter_type = adapter_type
        self.hexlog = MemoryHexLog()

    def identify(self) -> str:
        conn = self._factory(avensis.ECM.tx_id, avensis.ECM.rx_id, self.hexlog)
        with Client(conn, request_timeout=2.0, config=_uds_config()) as client:
            resp = client.read_data_by_identifier(avensis.VIN_DID)
            raw = resp.service_data.values[avensis.VIN_DID]
            if isinstance(raw, bytes):
                return raw.decode("ascii", errors="replace").strip("\x00")
            return str(raw)

    def scan(self) -> ScanResult:
        self.hexlog = MemoryHexLog()
        vin = ""
        codes: list[str] = []
        live: list[dict[str, Any]] = []
        modules: list[dict[str, Any]] = []

        for module in avensis.MODULES:
            timeout = 2.0 if module.name == "ECM" else 0.35
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
                with Client(conn, request_timeout=timeout, config=_uds_config(timeout)) as client:
                    node_codes = _read_dtcs(client)
                    info["dtcs"] = node_codes
                    info["reachable"] = True
                    if module.name == "ECM":
                        vin = _read_vin(client)
                        codes.extend(node_codes)
                        live.extend(_read_live(client))
                    else:
                        codes.extend(node_codes)
            except TimeoutException:
                info["error"] = "timeout"
            except TimeoutError:
                info["error"] = "timeout"
            except NegativeResponseException:
                info["reachable"] = True
                info["error"] = "nrc"
            modules.append(info)

        freeze = {item["name"]: item["value"] for item in live}
        classes = classify_codes(codes)
        return ScanResult(
            vin=vin,
            profile=avensis.PROFILE_ID,
            make=avensis.MAKE,
            model=avensis.MODEL,
            year=avensis.YEAR,
            adapter_type=self.adapter_type,
            protocol="uds_isotp_can",
            active_codes=codes,
            live=live,
            freeze_frame=freeze,
            raw_hex_stream=list(self.hexlog.lines),
            modules=modules,
            circuit_classes=classes,
            network=network_hint(modules),
        )

    def stream_dids(self, seconds: float = 6.0) -> dict[str, Any]:
        """Sample ECM DIDs for a wiggle test. No IO-control IDs are captured on this profile."""
        if seconds <= 0:
            seconds = 6.0
        if seconds > 20:
            seconds = 20.0
        if self.adapter_type == "mock":
            return _mock_stream()
        self.hexlog = MemoryHexLog()
        samples: list[dict[str, Any]] = []
        import time

        deadline = time.monotonic() + seconds
        conn = self._factory(avensis.ECM.tx_id, avensis.ECM.rx_id, self.hexlog)
        with Client(conn, request_timeout=1.0, config=_uds_config(1.0)) as client:
            while time.monotonic() < deadline:
                row: dict[str, Any] = {"t": round(seconds - (deadline - time.monotonic()), 2), "values": {}}
                for item in avensis.DIDS:
                    try:
                        resp = client.read_data_by_identifier(item.did)
                        raw = resp.service_data.values[item.did]
                        if isinstance(raw, (bytes, bytearray)):
                            row["values"][item.name] = _decode_scaled(bytes(raw), item.size, item.scale, item.offset)
                    except (NegativeResponseException, TimeoutException, TimeoutError):
                        row["values"][item.name] = None
                samples.append(row)
                time.sleep(0.35)
        return {
            "seconds": seconds,
            "module": "ECM",
            "tx_id": f"0x{avensis.ECM.tx_id:03X}",
            "io_control": "none_captured",
            "samples": samples,
            "raw_hex_stream": list(self.hexlog.lines),
        }


def _uds_config(timeout: float = 2.0) -> dict[str, Any]:
    cfg = dict(default_client_config)
    cfg["data_identifiers"] = {
        avensis.VIN_DID: RemainingCodec(),
        **{item.did: RemainingCodec() for item in avensis.DIDS},
    }
    cfg["p2_timeout"] = timeout
    cfg["request_timeout"] = timeout + 1.0
    return cfg


def _read_vin(client: Client) -> str:
    resp = client.read_data_by_identifier(avensis.VIN_DID)
    raw = resp.service_data.values[avensis.VIN_DID]
    if isinstance(raw, bytes):
        return raw.decode("ascii", errors="replace").strip("\x00")
    return str(raw)


def _read_dtcs(client: Client) -> list[str]:
    resp = client.get_dtc_by_status_mask(0xFF)
    if resp is None:
        return []
    out: list[str] = []
    for dtc in resp.service_data.dtcs:
        if hasattr(dtc, "id_iso"):
            out.append(dtc.id_iso().split("-")[0])
            continue
        high = (dtc.id >> 16) & 0xFF
        low = (dtc.id >> 8) & 0xFF
        out.append(decode_dtc(high, low))
    return out


def _read_live(client: Client) -> list[dict[str, Any]]:
    live: list[dict[str, Any]] = []
    for item in avensis.DIDS:
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
        except (NegativeResponseException, TimeoutException, TimeoutError):
            continue
    return live


def _decode_scaled(raw: bytes, size: int, scale: float, offset: float) -> float:
    chunk = raw[:size] if size else raw
    n = int.from_bytes(chunk, "big")
    return round(n * scale + offset, 3)


def _mock_stream() -> dict[str, Any]:
    # Wiggle story: actual angle drops out while target stays put (connector/loom).
    samples = []
    actuals = [0.0, 0.0, 12.4, 0.0, 0.0, 12.5, 0.0, 0.0]
    for i, actual in enumerate(actuals):
        samples.append(
            {
                "t": round(i * 0.4, 2),
                "values": {
                    "valvematic_target_angle": 12.5,
                    "valvematic_actual_angle": actual,
                    "engine_rpm": 1820,
                    "coolant_temp": 92.0,
                    "system_voltage": 13.8 if actual > 0 else 11.2,
                },
            }
        )
    return {
        "seconds": 3.2,
        "module": "ECM",
        "tx_id": "0x7E0",
        "io_control": "none_captured",
        "samples": samples,
        "raw_hex_stream": [],
    }


def openport_factory(lib, channel):
    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> J2534IsoTpConnection:
        return J2534IsoTpConnection(lib, channel, tx_id, rx_id, hexlog)

    return factory


def mock_factory():
    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> MockIsoTpConnection:
        replies = avensis.mock_replies(tx_id)
        silent = replies is None
        return MockIsoTpConnection(
            replies or {},
            hexlog,
            name=f"mock:{tx_id:03X}",
            silent=silent,
        )

    return factory

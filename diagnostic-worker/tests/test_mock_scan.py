from mechazone_worker.circuit import classify_code, network_hint
from mechazone_worker.session import DiagnosticSession, mock_factory
from mechazone_worker.transport import MemoryHexLog, ScriptedEcu


def test_mock_identify_and_scan() -> None:
    session = DiagnosticSession(mock_factory("avensis_3zr_fae"), "mock")
    ident = session.identify()
    vin = ident["vin"]
    assert vin == "SB1KV56E40E012345"
    assert ident["profile"] == "avensis_3zr_fae"
    result = session.scan()
    assert result.vin == vin
    assert "P1047" in result.active_codes
    assert "U011B" in result.active_codes
    names = {row["name"] for row in result.live}
    assert "valvematic_actual_angle" in names
    assert result.freeze_frame["valvematic_actual_angle"] == 0.0
    assert result.freeze_frame["valvematic_target_angle"] == 12.5
    assert any(line.startswith("TX") for line in result.raw_hex_stream)
    by_name = {m["name"]: m for m in result.modules}
    assert by_name["ECM"]["reachable"] is True
    assert by_name["VALVEMATIC"]["reachable"] is False
    assert by_name["VALVEMATIC"]["confirmed"] is True
    assert result.network["reading"] == "branch"
    classes = {row["code"]: row["class"] for row in result.circuit_classes}
    assert classes["U011B"] == "lost_communication"


def test_scan_upgrades_from_ecu_vin_without_identify() -> None:
    session = DiagnosticSession(mock_factory("avensis_3zr_fae"), "mock")
    result = session.scan()
    assert result.profile == "avensis_3zr_fae"
    assert any(m["name"] == "VALVEMATIC" for m in result.modules)
    names = {row["name"] for row in result.live}
    assert "valvematic_actual_angle" in names


def test_scan_overlay_year_from_decode() -> None:
    session = DiagnosticSession(mock_factory("avensis_3zr_fae"), "mock")
    session.identify()
    result = session.scan(make="Toyota", model="Avensis", year=2010)
    assert result.profile == "avensis_3zr_fae"
    assert result.year == 2010
    assert result.model == "Avensis"


def test_wiggle_stream_mock() -> None:
    session = DiagnosticSession(mock_factory("avensis_3zr_fae"), "mock")
    session.identify()
    stream = session.stream_dids(6)
    assert stream["io_control"] == "none_captured"
    assert len(stream["samples"]) >= 4
    actuals = [s["values"]["valvematic_actual_angle"] for s in stream["samples"]]
    assert 0.0 in actuals and max(actuals) >= 12.0


def test_scan_dark_bus_with_typed_vin() -> None:
    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del tx_id, rx_id
        return ScriptedEcu({}, hexlog, name="silent", silent=True)

    session = DiagnosticSession(factory, "mock")
    ident = session.identify()
    assert ident["vin"] == ""
    assert any("F190" in g for g in ident["coverage"]["gaps"])
    typed = session.identify(vin="JTDKB20E503123456")
    assert typed["vin"] == ""
    assert typed["profile"] == "toyota_common"
    result = session.scan(vin="JTDKB20E503123456")
    assert result.profile == "toyota_common"
    assert result.vin == "JTDKB20E503123456"
    assert result.modules
    assert all(not m["reachable"] for m in result.modules)


def test_identify_empty_isotp_payload_is_dark_not_crash() -> None:
    """J2534 START_OF_MESSAGE looks like an empty UDS frame to udsoncan."""

    class EmptyPdu(ScriptedEcu):
        def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
            self._hexlog.record("TX", payload)
            self.rxqueue.put(b"")

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> EmptyPdu:
        del tx_id, rx_id
        return EmptyPdu({}, hexlog, name="empty")

    session = DiagnosticSession(factory, "openport2_rev_e")
    ident = session.identify()
    assert ident["vin"] == ""
    assert any("F190" in g for g in ident["coverage"]["gaps"])


def test_classify_and_network() -> None:
    assert classify_code("U0073")["class"] == "bus_off"
    assert classify_code("P0113", "IAT circuit high")["class"] == "short_to_battery"
    hint = network_hint(
        [
            {"name": "ECM", "reachable": False, "confirmed": True},
            {"name": "ABS", "reachable": False, "confirmed": False},
        ]
    )
    assert hint["reading"] == "backbone"

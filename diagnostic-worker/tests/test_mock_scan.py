from mechazone_worker.circuit import classify_code, network_hint
from mechazone_worker.session import DiagnosticSession, mock_factory


def test_mock_identify_and_scan() -> None:
    session = DiagnosticSession(mock_factory(), "mock")
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


def test_wiggle_stream_mock() -> None:
    session = DiagnosticSession(mock_factory(), "mock")
    stream = session.stream_dids(6)
    assert stream["io_control"] == "none_captured"
    assert len(stream["samples"]) >= 4
    actuals = [s["values"]["valvematic_actual_angle"] for s in stream["samples"]]
    assert 0.0 in actuals and max(actuals) >= 12.0


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

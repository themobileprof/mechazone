from mechazone_worker.session import DiagnosticSession, mock_factory


def test_mock_identify_and_scan() -> None:
    session = DiagnosticSession(mock_factory(), "mock")
    vin = session.identify()
    assert vin == "SB1KV56E40E012345"
    result = session.scan()
    assert result.vin == vin
    assert "P1047" in result.active_codes
    assert "U011B" in result.active_codes
    names = {row["name"] for row in result.live}
    assert "valvematic_actual_angle" in names
    assert result.freeze_frame["valvematic_actual_angle"] == 0.0
    assert result.freeze_frame["valvematic_target_angle"] == 12.5
    assert any(line.startswith("TX") for line in result.raw_hex_stream)

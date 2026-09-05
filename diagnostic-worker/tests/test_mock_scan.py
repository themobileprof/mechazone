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


def test_clear_dtcs_mock() -> None:
    session = DiagnosticSession(mock_factory("avensis_3zr_fae"), "mock")
    scanned = session.scan()
    assert "P1047" in scanned.active_codes
    result = session.clear_dtcs()
    assert result["service"] == "0x14"
    assert result["group"] == "0xFFFFFF"
    assert "P1047" in result["codes_before"]
    assert "U011B" in result["codes_before"]
    assert result["codes_after"] == []
    by_name = {m["name"]: m for m in result["modules"]}
    assert by_name["ECM"]["cleared"] is True
    assert by_name["ECM"]["attempted"] is True
    assert by_name["VALVEMATIC"]["attempted"] is False
    assert any("14 FF FF FF" in line for line in result["raw_hex_stream"])
    try:
        session.clear_dtcs()
        raise AssertionError("second clear should find no codes")
    except RuntimeError as exc:
        assert "codes" in str(exc).lower()


def test_clear_dtcs_extended_session_then_clears() -> None:
    from mechazone_worker.profiles.avensis_3zr_fae import mock_replies

    class NeedsExtended(ScriptedEcu):
        def specific_send(self, payload: bytes, timeout: float | None = None) -> None:
            self._hexlog.record("TX", payload)
            key = bytes(payload)
            sid = key[0] if key else 0x00
            extended = getattr(self, "_extended", False)
            if sid == 0x10:
                resp = b"\x50\x03\x00\x32\x01\xf4" if key == b"\x10\x03" else bytes([0x7F, 0x10, 0x12])
                if key == b"\x10\x03":
                    self._extended = True
            elif sid == 0x14:
                if extended:
                    self._dtcs_cleared = True
                    resp = b"\x54"
                else:
                    resp = bytes([0x7F, 0x14, 0x7F])
            elif getattr(self, "_dtcs_cleared", False) and sid == 0x19:
                rest = key[1:] if len(key) > 1 else b"\x02\xff"
                resp = bytes([0x59]) + rest
            else:
                resp = self.ResponseData.get(key) or bytes([0x7F, sid, 0x11])
            self._hexlog.record("RX", resp)
            self.rxqueue.put(resp)

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del rx_id
        replies = mock_replies(tx_id)
        if replies is None:
            return ScriptedEcu({}, hexlog, silent=True)
        return NeedsExtended(replies, hexlog, tx_id=tx_id)

    session = DiagnosticSession(factory, "mock")
    session.scan()
    result = session.clear_dtcs()
    ecm = next(m for m in result["modules"] if m["name"] == "ECM")
    assert ecm["cleared"] is True
    assert ecm["session"] == "extended"
    assert result["codes_after"] == []
    assert any("10 03" in line for line in result["raw_hex_stream"])


def test_clear_dtcs_falls_back_to_emissions_group() -> None:
    from mechazone_worker.profiles.avensis_3zr_fae import mock_replies

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del rx_id
        replies = mock_replies(tx_id)
        if replies is None:
            return ScriptedEcu({}, hexlog, silent=True)
        pinned = dict(replies)
        pinned[bytes([0x14, 0xFF, 0xFF, 0xFF])] = bytes([0x7F, 0x14, 0x31])
        return ScriptedEcu(pinned, hexlog, tx_id=tx_id)

    session = DiagnosticSession(factory, "mock")
    session.scan()
    result = session.clear_dtcs()
    ecm = next(m for m in result["modules"] if m["name"] == "ECM")
    assert ecm["cleared"] is True
    assert ecm["group"] == "0x000000"
    assert result["codes_after"] == []


def test_clear_dtcs_nrc_does_not_retry_security() -> None:
    from mechazone_worker.profiles.avensis_3zr_fae import mock_replies

    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del rx_id
        replies = mock_replies(tx_id)
        if replies is None:
            return ScriptedEcu({}, hexlog, silent=True)
        pinned = dict(replies)
        pinned[bytes([0x14, 0xFF, 0xFF, 0xFF])] = bytes([0x7F, 0x14, 0x22])
        return ScriptedEcu(pinned, hexlog)

    session = DiagnosticSession(factory, "mock")
    session.scan()
    result = session.clear_dtcs()
    ecm = next(m for m in result["modules"] if m["name"] == "ECM")
    assert ecm["cleared"] is False
    assert ecm["attempted"] is True
    assert "P1047" in ecm["codes_after"]
    assert "nrc" in (ecm.get("error") or "")
    assert "seed" in (ecm.get("gap") or "").lower()
    assert "P1047" in result["codes_after"]


def test_clear_dtcs_skips_dark_nodes() -> None:
    def factory(tx_id: int, rx_id: int, hexlog: MemoryHexLog) -> ScriptedEcu:
        del tx_id, rx_id
        return ScriptedEcu({}, hexlog, name="silent", silent=True)

    session = DiagnosticSession(factory, "mock")
    session.scan(vin="JTDKB20E503123456")
    try:
        session.clear_dtcs()
        raise AssertionError("dark bus must not send $14")
    except RuntimeError as exc:
        assert "$14" in str(exc) or "codes" in str(exc).lower()


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

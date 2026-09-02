from mechazone_worker.profiles import select_profile
from mechazone_worker.profiles.avensis_3zr_fae import MOCK_VIN
from mechazone_worker.session import DiagnosticSession, generic_mock_factory


def test_select_avensis_mock_vin() -> None:
    p = select_profile(MOCK_VIN)
    assert p.id == "avensis_3zr_fae"
    assert p.depth == "captured"


def test_select_avensis_from_model_hint() -> None:
    p = select_profile("JTDKB20E503123456", make="Toyota", model="Avensis", year=2010)
    assert p.id == "avensis_3zr_fae"


def test_select_toyota_probe() -> None:
    p = select_profile("JTDKB20E503123456")
    assert p.id == "toyota_common"
    assert p.depth == "toyota_probe"
    assert any("probe" in g.lower() for g in p.coverage()["gaps"])


def test_select_generic_honda() -> None:
    p = select_profile("1HGCM82633A004352")
    assert p.id == "generic_uds"
    assert p.depth == "iso_15765_4"


def test_select_tesla_gap() -> None:
    p = select_profile("5YJSA1E14HF123456")
    assert p.id == "generic_uds"
    gaps = " ".join(p.coverage()["gaps"])
    assert "Tesla" in gaps


def test_select_byd_gap() -> None:
    p = select_profile("LGXCE6CB0N0123456")
    gaps = " ".join(p.coverage()["gaps"])
    assert "BMS" in gaps


def test_generic_mock_scan() -> None:
    session = DiagnosticSession(generic_mock_factory(), "openport2_rev_e")
    ident = session.identify()
    assert ident["vin"].startswith("1HG")
    assert ident["profile"] == "generic_uds"
    result = session.scan(make="Honda", model="Accord", year=2003)
    assert result.profile == "generic_uds"
    assert result.make == "Honda"
    assert result.live == []
    assert any(m["name"] == "ECM" and m["reachable"] for m in result.modules)
    assert "P0113" in result.active_codes
    assert result.coverage["depth"] == "iso_15765_4"
    wiggle = session.stream_dids(2)
    assert wiggle["samples"] == []
    assert wiggle.get("gap")

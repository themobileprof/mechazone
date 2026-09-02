from mechazone_worker.detect import detect_adapters


def test_detect_recommends_openport_when_usb_present() -> None:
    out = detect_adapters(usb=[("0403", "cc4d")])
    assert out["recommended"] == "openport2_rev_e"
    by_id = {d["id"]: d for d in out["devices"]}
    assert by_id["openport2_rev_e"]["present"] is True
    assert by_id["openport2_rev_e"]["connectable"] is True
    assert by_id["mock"]["recommended"] is False


def test_detect_elm_is_not_connectable() -> None:
    out = detect_adapters(usb=[("1a86", "7523")])
    by_id = {d["id"]: d for d in out["devices"]}
    assert by_id["elm327"]["connectable"] is False
    assert by_id["elm327"]["capability"] == "detect_only"
    assert out["recommended"] == "mock"


def test_detect_unknown_usb_is_listed() -> None:
    out = detect_adapters(usb=[("1234", "abcd")])
    labels = [d["label"] for d in out["devices"]]
    assert any("1234:abcd" in lab for lab in labels)

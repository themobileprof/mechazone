from mechazone_worker.hexutil import decode_dtc, from_hex, to_hex


def test_hex_roundtrip() -> None:
    assert to_hex(from_hex("22 F1 90")) == "22 F1 90"


def test_dtc_p1047() -> None:
    assert decode_dtc(0x10, 0x47) == "P1047"


def test_dtc_u011b() -> None:
    assert decode_dtc(0xC1, 0x1B) == "U011B"

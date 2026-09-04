from mechazone_worker.j2534 import completed_uds_payload


def test_start_of_message_is_not_a_uds_reply() -> None:
    can_id = bytes.fromhex("000007e8")
    assert completed_uds_payload(0x02, 4, can_id) is None


def test_tx_indication_is_not_a_uds_reply() -> None:
    can_id = bytes.fromhex("000007e0")
    assert completed_uds_payload(0x08, 4, can_id) is None


def test_tx_echo_is_not_a_uds_reply() -> None:
    frame = bytes.fromhex("000007e0") + bytes.fromhex("22f190")
    assert completed_uds_payload(0x01, len(frame), frame) is None


def test_completed_vin_pdu() -> None:
    vin = b"SB1KV56E40E012345"
    frame = bytes.fromhex("000007e8") + bytes([0x62, 0xF1, 0x90]) + vin
    payload = completed_uds_payload(0x00, len(frame), frame)
    assert payload == bytes([0x62, 0xF1, 0x90]) + vin


def test_empty_data_after_can_id() -> None:
    frame = bytes.fromhex("000007e8")
    assert completed_uds_payload(0x00, 4, frame) is None

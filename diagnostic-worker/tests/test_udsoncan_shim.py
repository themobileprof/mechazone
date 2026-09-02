def test_udsoncan_j2534_imports_after_linux_shim() -> None:
    import mechazone_worker  # noqa: F401 — maps WINFUNCTYPE on Linux
    from udsoncan.connections import FakeConnection
    from udsoncan.j2534 import J2534, Protocol_ID

    assert Protocol_ID.ISO15765.value == 6
    assert J2534 is not None
    assert FakeConnection is not None

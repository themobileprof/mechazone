package pii

import "testing"

func TestRejectCloudPII(t *testing.T) {
	if err := RejectCloudPII([]byte(`{"vin":"SB1KV56E40E012345","mileage_km":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := RejectCloudPII([]byte(`{"vin":"X","customer_name":"Ada"}`)); err == nil {
		t.Fatal("expected pii rejection")
	}
	if err := RejectCloudPII([]byte(`{"plate":"ABC-123"}`)); err == nil {
		t.Fatal("expected plate rejection")
	}
}

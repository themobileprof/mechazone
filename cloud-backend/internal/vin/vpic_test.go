package vin

import "testing"

func TestNormalize(t *testing.T) {
	got, err := Normalize(" sb1kv56e40e012345 ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "SB1KV56E40E012345" {
		t.Fatalf("got %s", got)
	}
	if _, err := Normalize("SHORT"); err == nil {
		t.Fatal("expected length error")
	}
	if _, err := Normalize("SB1KV56E40E01234I"); err == nil {
		t.Fatal("expected forbidden I")
	}
}

package ledger

import (
	"context"
	"testing"
	"time"

	"mechazone/cloud-backend/internal/vin"
)

func TestClipRunes(t *testing.T) {
	if clipRunes("abc", 5) != "abc" {
		t.Fatal("short")
	}
	if got := clipRunes("abcdefghij", 4); got != "abcd" {
		t.Fatalf("got %q", got)
	}
}

func TestShopCustomerRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	vin := "ZZZZCUSTDEV000001"
	if err := s.EnsureVehicle(ctx, vin, "Toyota", "Unknown", 2010, "test"); err != nil {
		t.Fatal(err)
	}
	shop := "00000000-0000-4000-8000-000000000001"
	tech := "00000000-0000-4000-8000-000000000002"
	saved, err := s.UpsertShopCustomer(ctx, vin, shop, tech, ShopCustomer{
		DisplayName: "Ada Okonkwo", Phone: "0803", Plate: "abc-123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Plate != "ABC-123" {
		t.Fatalf("plate %q", saved.Plate)
	}
	got, err := s.ShopCustomer(ctx, vin, shop, tech)
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Ada Okonkwo" || got.Plate != "ABC-123" {
		t.Fatalf("%+v", got)
	}
	other, err := s.ShopCustomer(ctx, vin, "00000000-0000-4000-8000-000000000099", tech)
	if err != nil {
		t.Fatal(err)
	}
	if other.DisplayName != "" {
		t.Fatal("another shop must not see this name")
	}
}

func TestSaveVINDecodeFillsUnknownBody(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	code := "ZZZZDECODEFAAA001"
	if err := s.EnsureVehicle(ctx, code, "Toyota", "Unknown", 0, "vpic"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveVINDecode(ctx, vin.Decode{
		VIN: code, Make: "Toyota", Model: "Avensis", Year: 2010, Source: "vpic", Raw: []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}
	h, err := s.History(ctx, code, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	if h.Vehicle == nil || h.Vehicle.Model != "Avensis" || h.Vehicle.Year != 2010 {
		t.Fatalf("vehicle %+v", h.Vehicle)
	}
}

func TestApplyPlateIfBlankFillsUnknownYear(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	code := "2T1BR32EX7C733665"
	if err := s.EnsureVehicle(ctx, code, "Unknown", "Unknown", 0, "vpic"); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyPlateIfBlank(ctx, code, vin.ReadPlate(code)); err != nil {
		t.Fatal(err)
	}
	h, err := s.History(ctx, code, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	if h.Vehicle == nil || h.Vehicle.Make != "Toyota" || h.Vehicle.Year != 2007 {
		t.Fatalf("vehicle %+v", h.Vehicle)
	}
	if h.Vehicle.Model != "Unknown" {
		t.Fatalf("must not invent a body: %+v", h.Vehicle)
	}
}

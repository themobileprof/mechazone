package vin

import "testing"

func TestReadPlateToyotaCanadaCorollaClass(t *testing.T) {
	p := ReadPlate("2T1BR32EX7C733665")
	if p.WMI != "2T1" || p.Maker != "Toyota" || p.Region != "Canada" || p.Kind != "passenger" {
		t.Fatalf("wmi %+v", p)
	}
	if p.Year != 2007 || p.YearCode != "7" {
		t.Fatalf("year %+v", p)
	}
	if p.Plant != "C" {
		t.Fatalf("plant %s", p.Plant)
	}
}

func TestReadPlateHondaUSA(t *testing.T) {
	p := ReadPlate("1HGCM82633A004352")
	if p.Maker != "Honda" || p.Region != "USA" || p.Year != 2003 {
		t.Fatalf("%+v", p)
	}
}

func TestReadPlateUnknownYearDigit(t *testing.T) {
	p := ReadPlate("SB1BL75L60E023110")
	if p.Maker != "Toyota" || p.Region != "UK" {
		t.Fatalf("maker %+v", p)
	}
	if p.Year != 0 || p.YearCode != "0" {
		t.Fatalf("year digit 0 is not SAE; got %+v", p)
	}
}

func TestEnrichEmptyDoesNotInventModel(t *testing.T) {
	dec := Decode{VIN: "2T1BR32EX7C733665", Source: "vpic", Empty: true}
	EnrichEmpty(&dec, ReadPlate(dec.VIN))
	if dec.Make != "Toyota" || dec.Year != 2007 {
		t.Fatalf("%+v", dec)
	}
	if dec.Model != "" {
		t.Fatalf("must not store a class as model: %q", dec.Model)
	}
	if dec.Source != "vin_plate" {
		t.Fatalf("source %s", dec.Source)
	}
}

func TestEnrichEmptyLeavesNamedDecode(t *testing.T) {
	dec := Decode{VIN: "1HGCM82633A004352", Make: "Honda", Model: "Accord", Year: 2003, Source: "vpic"}
	EnrichEmpty(&dec, ReadPlate(dec.VIN))
	if dec.Make != "Honda" || dec.Model != "Accord" || dec.Year != 2003 || dec.Source != "vpic" {
		t.Fatalf("%+v", dec)
	}
}

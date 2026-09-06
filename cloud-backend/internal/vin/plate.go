package vin

import (
	"strings"
	"time"
)

// Plate is ISO 3779 structure from the 17 characters. Used when vPIC is empty.
// It is not a model encyclopedia and must not overwrite a named vPIC body.
type Plate struct {
	WMI      string `json:"wmi"`
	YearCode string `json:"year_code"`
	Year     int    `json:"year,omitempty"`
	Plant    string `json:"plant"`
	Maker    string `json:"maker,omitempty"`
	Region   string `json:"region,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

type wmiHint struct {
	maker  string
	region string
	kind   string
}

var wmiTable = map[string]wmiHint{
	"2T1": {maker: "Toyota", region: "Canada", kind: "passenger"},
	"2T2": {maker: "Lexus", region: "Canada", kind: "passenger"},
	"2T3": {maker: "Toyota", region: "Canada", kind: ""},
	"3TM": {maker: "Toyota", region: "Mexico", kind: ""},
	"4T1": {maker: "Toyota", region: "USA", kind: "passenger"},
	"4T3": {maker: "Toyota", region: "USA", kind: ""},
	"4T4": {maker: "Toyota", region: "USA", kind: ""},
	"4TA": {maker: "Toyota", region: "USA", kind: ""},
	"5TD": {maker: "Toyota", region: "USA", kind: ""},
	"5TE": {maker: "Toyota", region: "USA", kind: ""},
	"5TF": {maker: "Toyota", region: "USA", kind: ""},
	"JTD": {maker: "Toyota", region: "Japan", kind: ""},
	"JTE": {maker: "Toyota", region: "Japan", kind: ""},
	"JTN": {maker: "Toyota", region: "Japan", kind: ""},
	"JT2": {maker: "Toyota", region: "Japan", kind: "passenger"},
	"JT3": {maker: "Toyota", region: "Japan", kind: ""},
	"JT4": {maker: "Toyota", region: "Japan", kind: ""},
	"JT5": {maker: "Toyota", region: "Japan", kind: ""},
	"JT6": {maker: "Toyota", region: "Japan", kind: ""},
	"JT8": {maker: "Lexus", region: "Japan", kind: ""},
	"SB1": {maker: "Toyota", region: "UK", kind: "passenger"},
	"AHT": {maker: "Toyota", region: "South Africa", kind: ""},
	"NMT": {maker: "Toyota", region: "Turkey", kind: ""},
	"MR0": {maker: "Toyota", region: "Thailand", kind: ""},
	"MR2": {maker: "Toyota", region: "Thailand", kind: ""},
	"MM7": {maker: "Toyota", region: "Thailand", kind: ""},
	"MM8": {maker: "Toyota", region: "Thailand", kind: ""},
	"PN1": {maker: "Toyota", region: "Malaysia", kind: ""},
	"PN4": {maker: "Toyota", region: "Malaysia", kind: ""},
	"JTH": {maker: "Lexus", region: "Japan", kind: ""},
	"JTJ": {maker: "Lexus", region: "Japan", kind: ""},
	"1HG": {maker: "Honda", region: "USA", kind: "passenger"},
	"2HG": {maker: "Honda", region: "Canada", kind: "passenger"},
	"19X": {maker: "Honda", region: "USA", kind: "passenger"},
	"JHM": {maker: "Honda", region: "Japan", kind: "passenger"},
	"SHH": {maker: "Honda", region: "UK", kind: ""},
}

const yearCycle = "ABCDEFGHJKLMNPRSTVWXY123456789"

// ReadPlate parses WMI / year digit / plant. Year uses the latest SAE cycle not in the future.
func ReadPlate(vin string) Plate {
	v := strings.ToUpper(strings.TrimSpace(vin))
	if len(v) != 17 {
		return Plate{}
	}
	p := Plate{
		WMI:      v[:3],
		YearCode: string(v[9]),
		Plant:    string(v[10]),
	}
	if h, ok := wmiTable[p.WMI]; ok {
		p.Maker = h.maker
		p.Region = h.region
		p.Kind = h.kind
	}
	p.Year = modelYear(p.YearCode, time.Now().Year())
	return p
}

func modelYear(code string, now int) int {
	if len(code) != 1 {
		return 0
	}
	i := strings.IndexRune(yearCycle, rune(code[0]))
	if i < 0 {
		return 0
	}
	capYear := now + 1
	best := 0
	for _, base := range []int{1980, 2010, 2040} {
		y := base + i
		if y <= capYear && y > best {
			best = y
		}
	}
	return best
}

// EnrichEmpty fills make/year from the plate when an API decode left them blank.
// Model stays empty so a WMI class is not stored as a named body.
func EnrichEmpty(dec *Decode, plate Plate) {
	if dec == nil {
		return
	}
	if strings.TrimSpace(dec.Make) == "" && plate.Maker != "" {
		dec.Make = plate.Maker
		if dec.Source == "" || dec.Source == "vpic" {
			dec.Source = "vin_plate"
		}
	}
	if dec.Year == 0 && plate.Year > 0 {
		dec.Year = plate.Year
		if dec.Source == "" || dec.Source == "vpic" {
			dec.Source = "vin_plate"
		}
	}
}

// Useful reports whether the plate has anything the bay should persist.
func (p Plate) Useful() bool {
	return p.Maker != "" || p.Year > 0
}

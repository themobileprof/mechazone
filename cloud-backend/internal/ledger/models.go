package ledger

import (
	"encoding/json"
	"time"
)

type Vehicle struct {
	VIN          string    `json:"vin"`
	Make         string    `json:"make"`
	Model        string    `json:"model"`
	Year         int       `json:"manufacture_year"`
	DecodeSource string    `json:"decode_source"`
	FirstSeenAt  time.Time `json:"first_seen_at"`
}

// Session is one visit: a live OpenPort scan or an attached vendor report.
type Session struct {
	ID            string          `json:"id"`
	VIN           string          `json:"vin"`
	ShopID        string          `json:"shop_id"`
	TechnicianID  string          `json:"technician_id"`
	Mileage       int             `json:"mileage_km"`
	AdapterType   string          `json:"adapter_type"`
	HostOS        string          `json:"host_os"`
	Protocol      string          `json:"protocol"`
	ActiveCodes   []string        `json:"active_codes"`
	FreezeFrame   json.RawMessage `json:"freeze_frame"`
	RawHexExcerpt string          `json:"raw_hex_excerpt,omitempty"`
	Outcome       string          `json:"outcome"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Resolution is the technician's closeout: what was done, not a public VIN file.
type Resolution struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	VIN           string    `json:"vin"`
	TechnicianID  string    `json:"technician_id"`
	DTC           string    `json:"diagnostic_trouble_code"`
	RootCause     string    `json:"root_cause_explanation"`
	PartsReplaced []string  `json:"parts_replaced"`
	Verified      bool      `json:"is_verified_fix"`
	CreatedAt     time.Time `json:"created_at"`
}

type DTC struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Source   string `json:"source"`
}

// JobImport is metadata for a file attached as adapter_type=imported_report.
type JobImport struct {
	Source       string `json:"source"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	ByteSize     int    `json:"byte_size"`
	Note         string `json:"note"`
}

type Job struct {
	SessionID      string     `json:"session_id"`
	CreatedAt      time.Time  `json:"created_at"`
	MileageKM      int        `json:"mileage_km"`
	TechnicianName string     `json:"technician_name"`
	TechnicianID   string     `json:"technician_id"`
	Outcome        string     `json:"outcome"`
	ActiveCodes    []string   `json:"active_codes"`
	Work           string     `json:"work"`
	PartsReplaced  []string   `json:"parts_replaced"`
	VerifiedFix    bool       `json:"verified_fix"`
	ResolutionID   string     `json:"resolution_id,omitempty"`
	CloseoutCode   string     `json:"closeout_code,omitempty"`
	AdapterType    string     `json:"adapter_type"`
	Protocol       string     `json:"protocol"`
	Import         *JobImport `json:"import,omitempty"`
}

// History is this shop's (or freelancer's) jobs on a VIN — never another shop's file.
type History struct {
	Vehicle     *Vehicle     `json:"vehicle"`
	FirstSeen   bool         `json:"first_seen"`
	Jobs        []Job        `json:"jobs"`
	Sessions    []Session    `json:"sessions"`
	Resolutions []Resolution `json:"resolutions"`
}

// SessionIngest is the scan body. ShopID and TechnicianID in JSON are ignored; handlers copy them from the cookie.
type SessionIngest struct {
	VIN          string          `json:"vin"`
	ShopID       string          `json:"shop_id"`
	TechnicianID string          `json:"technician_id"`
	MileageKM    int             `json:"mileage_km"`
	AdapterType  string          `json:"adapter_type"`
	HostOS       string          `json:"host_os"`
	Protocol     string          `json:"protocol"`
	ActiveCodes  []string        `json:"active_codes"`
	FreezeFrame  json.RawMessage `json:"freeze_frame"`
	RawHexStream []string        `json:"raw_hex_stream"`
	CapturedAt   time.Time       `json:"captured_at"`
	MakeHint     string          `json:"make_hint,omitempty"`
	ModelHint    string          `json:"model_hint,omitempty"`
	YearHint     int             `json:"year_hint,omitempty"`
}

type Closeout struct {
	Outcome   string   `json:"outcome"`
	DTC       string   `json:"diagnostic_trouble_code"`
	RootCause string   `json:"root_cause_explanation"`
	Parts     []string `json:"parts_replaced"`
}

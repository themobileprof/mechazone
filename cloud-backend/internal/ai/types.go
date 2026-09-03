package ai

import "encoding/json"

type LiveRow struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	DID   string  `json:"did"`
}

// Request is the posted scan plus cookie-scoped ShopID/TechnicianID (json:"-").
type Request struct {
	VIN          string          `json:"vin"`
	SessionID    string          `json:"session_id"`
	Make         string          `json:"make"`
	Model        string          `json:"model"`
	Year         int             `json:"year"`
	EngineHint   string          `json:"engine_hint"`
	ActiveCodes  []string        `json:"active_codes"`
	Live         []LiveRow       `json:"live"`
	Modules      []ModuleHit     `json:"modules"`
	FreezeFrame  json.RawMessage `json:"freeze_frame"`
	AdapterType  string          `json:"adapter_type"`
	Protocol     string          `json:"protocol"`
	ShopID       string          `json:"-"`
	TechnicianID string          `json:"-"`
	Language     string          `json:"language"`
}

// Lookout is a risk from this shop's jobs or this scan. Evidence prefixes must pass Sanitize.
type Lookout struct {
	Text     string   `json:"text"`
	Evidence []string `json:"evidence"`
}

type Cause struct {
	Title       string   `json:"title"`
	Probability float64  `json:"probability"`
	Evidence    []string `json:"evidence"`
}

type Step struct {
	Order   int      `json:"order"`
	Kind    string   `json:"kind"`
	Title   string   `json:"title"`
	Detail  string   `json:"detail"`
	Pass    string   `json:"pass,omitempty"`
	Fail    string   `json:"fail,omitempty"`
	Adapter bool     `json:"adapter"`
	Figures []string `json:"figures,omitempty"`
}

// Playbook is what to test on this VIN. Uncited pins belong in Gaps; figures are retrieved IDs only.
type Playbook struct {
	VIN            string         `json:"vin"`
	Platform       string         `json:"platform"`
	Lookouts       []Lookout      `json:"lookouts"`
	LikelyCauses   []Cause        `json:"likely_causes"`
	Steps          []Step         `json:"steps"`
	Validation     string         `json:"validation"`
	Gaps           []string       `json:"gaps"`
	Model          string         `json:"model,omitempty"`
	FirstSeen      bool           `json:"first_seen"`
	CircuitClasses []CircuitClass `json:"circuit_classes,omitempty"`
	Network        NetworkHint    `json:"network,omitempty"`
	ManualFigures  []ManualFigure `json:"manual_figures,omitempty"`
}

type ManualFigure struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Page     int    `json:"page"`
	Caption  string `json:"caption"`
	Language string `json:"language"`
	ImageURL string `json:"image_url,omitempty"`
	OCRText  string `json:"ocr_text,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

package ai

import (
	"bytes"
	"encoding/json"
	"strings"

	"mechazone/cloud-backend/internal/ledger"
)

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
	SourceID     string          `json:"source_id,omitempty"`
	ShopID       string          `json:"-"`
	TechnicianID string          `json:"-"`
	Language     string          `json:"language"`
}

// StringList accepts a JSON array or a single string. Hosted models often emit evidence as a string.
type StringList []string

func (s *StringList) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*s = StringList{}
		return nil
	}
	if b[0] == '"' {
		var one string
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		one = strings.TrimSpace(one)
		if one == "" {
			*s = StringList{}
			return nil
		}
		*s = StringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*s = StringList(many)
	return nil
}

// Lookout is a risk from this shop's jobs or this scan. Evidence prefixes must pass Sanitize.
type Lookout struct {
	Text     string     `json:"text"`
	Evidence StringList `json:"evidence"`
}

type Cause struct {
	Title       string     `json:"title"`
	Probability float64    `json:"probability"`
	Evidence    StringList `json:"evidence"`
}

type Step struct {
	Order   int        `json:"order"`
	Kind    string     `json:"kind"`
	Title   string     `json:"title"`
	Detail  string     `json:"detail"`
	Pass    string     `json:"pass,omitempty"`
	Fail    string     `json:"fail,omitempty"`
	Adapter bool       `json:"adapter"`
	Figures StringList `json:"figures,omitempty"`
}

// Playbook is what to test on this VIN. Uncited pins belong in Gaps; figures are retrieved IDs only.
type Playbook struct {
	VIN             string                 `json:"vin"`
	Platform        string                 `json:"platform"`
	Lookouts        []Lookout              `json:"lookouts"`
	LikelyCauses    []Cause                `json:"likely_causes"`
	Steps           []Step                 `json:"steps"`
	Validation      string                 `json:"validation"`
	Gaps            StringList             `json:"gaps"`
	Model           string                 `json:"model,omitempty"`
	FirstSeen       bool                   `json:"first_seen"`
	Checks          []ledger.PlaybookCheck `json:"checks,omitempty"`
	CircuitClasses  []CircuitClass         `json:"circuit_classes,omitempty"`
	Network         NetworkHint            `json:"network,omitempty"`
	ManualFigures   []ManualFigure         `json:"manual_figures,omitempty"`
	Manual          *PinnedManual          `json:"manual,omitempty"`
	RetrievedChunks int                    `json:"retrieved_chunks,omitempty"`
}

type PinnedManual struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Make     string `json:"make"`
	Model    string `json:"model"`
	YearFrom int    `json:"year_from"`
	YearTo   int    `json:"year_to"`
	Engine   string `json:"engine"`
	Language string `json:"language"`
	Chunks   int    `json:"chunks"`
	Figures  int    `json:"figures"`
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

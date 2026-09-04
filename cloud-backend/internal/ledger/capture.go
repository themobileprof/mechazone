package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// BusModule is one observed UDS node. EverReachable is the mapping fact; Reachable is the last scan.
type BusModule struct {
	Name          string   `json:"name"`
	TxID          string   `json:"tx_id"`
	RxID          string   `json:"rx_id"`
	Family        string   `json:"family,omitempty"`
	Confirmed     bool     `json:"confirmed"`
	Reachable     bool     `json:"reachable"`
	EverReachable bool     `json:"ever_reachable"`
	DTCs          []string `json:"dtcs,omitempty"`
}

type BusIdentity struct {
	Name string `json:"name"`
	DID  string `json:"did"`
	Text string `json:"text"`
}

type BusLive struct {
	Name  string `json:"name"`
	DID   string `json:"did"`
	Unit  string `json:"unit,omitempty"`
	Value any    `json:"value,omitempty"`
}

// BusCapture is this shop's (or freelancer's) observed bus map on a VIN. Not a closeout.
type BusCapture struct {
	VIN           string          `json:"vin"`
	Profile       string          `json:"profile"`
	AdapterType   string          `json:"adapter_type"`
	HostOS        string          `json:"host_os"`
	Protocol      string          `json:"protocol"`
	MakeHint      string          `json:"make_hint,omitempty"`
	ModelHint     string          `json:"model_hint,omitempty"`
	YearHint      int             `json:"year_hint,omitempty"`
	Modules       []BusModule     `json:"modules"`
	Identity      []BusIdentity   `json:"identity"`
	Live          []BusLive       `json:"live"`
	ActiveCodes   []string        `json:"active_codes"`
	Coverage      json.RawMessage `json:"coverage,omitempty"`
	RawHexExcerpt string          `json:"raw_hex_excerpt,omitempty"`
	ScanCount     int             `json:"scan_count"`
	FirstSeenAt   time.Time       `json:"first_seen_at"`
	LastSeenAt    time.Time       `json:"last_seen_at"`
}

// BusCaptureIn is the bay payload. Shop and technician IDs come from the cookie.
type BusCaptureIn struct {
	Profile      string          `json:"profile"`
	AdapterType  string          `json:"adapter_type"`
	HostOS       string          `json:"host_os"`
	Protocol     string          `json:"protocol"`
	MakeHint     string          `json:"make_hint,omitempty"`
	ModelHint    string          `json:"model_hint,omitempty"`
	YearHint     int             `json:"year_hint,omitempty"`
	Modules      []BusModule     `json:"modules"`
	Identity     []BusIdentity   `json:"identity"`
	Live         []BusLive       `json:"live"`
	ActiveCodes  []string        `json:"active_codes"`
	Coverage     json.RawMessage `json:"coverage"`
	RawHexStream []string        `json:"raw_hex_stream"`
}

func canonID(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.TrimPrefix(s, "0X")
}

func moduleKey(m BusModule) string {
	return strings.ToUpper(strings.TrimSpace(m.Name)) + "|" + canonID(m.TxID)
}

func mergeBusModules(prev, next []BusModule) []BusModule {
	nextKeys := make(map[string]struct{}, len(next))
	for _, m := range next {
		if m.Name == "" && m.TxID == "" {
			continue
		}
		nextKeys[moduleKey(m)] = struct{}{}
	}
	by := make(map[string]BusModule, len(prev)+len(next))
	order := make([]string, 0, len(prev)+len(next))
	add := func(m BusModule) {
		if m.Name == "" && m.TxID == "" {
			return
		}
		m.TxID = canonID(m.TxID)
		m.RxID = canonID(m.RxID)
		if m.EverReachable || m.Reachable {
			m.EverReachable = true
		}
		k := moduleKey(m)
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		if old, ok := by[k]; ok {
			m.EverReachable = old.EverReachable || m.EverReachable || old.Reachable
			m.Confirmed = old.Confirmed || m.Confirmed
			if m.Family == "" {
				m.Family = old.Family
			}
			if m.RxID == "" {
				m.RxID = old.RxID
			}
		}
		by[k] = m
	}
	for _, m := range prev {
		add(m)
	}
	for _, m := range next {
		add(m)
	}
	out := make([]BusModule, 0, len(order))
	for _, k := range order {
		m := by[k]
		if _, ok := nextKeys[k]; !ok {
			m.Reachable = false
			m.DTCs = nil
		}
		out = append(out, m)
	}
	return out
}

func mergeBusIdentity(prev, next []BusIdentity) []BusIdentity {
	by := make(map[string]BusIdentity, len(prev)+len(next))
	order := make([]string, 0, len(prev)+len(next))
	add := func(row BusIdentity) {
		k := strings.ToUpper(strings.TrimSpace(row.DID))
		if k == "" {
			k = strings.ToLower(strings.TrimSpace(row.Name))
		}
		if k == "" {
			return
		}
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		old := by[k]
		if strings.TrimSpace(row.Text) == "" {
			row.Text = old.Text
		}
		if row.Name == "" {
			row.Name = old.Name
		}
		by[k] = row
	}
	for _, row := range prev {
		add(row)
	}
	for _, row := range next {
		add(row)
	}
	out := make([]BusIdentity, 0, len(order))
	for _, k := range order {
		out = append(out, by[k])
	}
	return out
}

func mergeBusLive(prev, next []BusLive) []BusLive {
	by := make(map[string]BusLive, len(prev)+len(next))
	order := make([]string, 0, len(prev)+len(next))
	add := func(row BusLive) {
		k := strings.ToUpper(strings.TrimSpace(row.DID))
		if k == "" {
			k = strings.ToLower(strings.TrimSpace(row.Name))
		}
		if k == "" {
			return
		}
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		old := by[k]
		if row.Name == "" {
			row.Name = old.Name
		}
		if row.Unit == "" {
			row.Unit = old.Unit
		}
		by[k] = row
	}
	for _, row := range prev {
		add(row)
	}
	for _, row := range next {
		add(row)
	}
	out := make([]BusLive, 0, len(order))
	for _, k := range order {
		out = append(out, by[k])
	}
	return out
}

func hexExcerpt(lines []string) string {
	excerpt := strings.Join(lines, "\n")
	if len(excerpt) > 8000 {
		return excerpt[:8000]
	}
	return excerpt
}

func (s *Store) BusCapture(ctx context.Context, vin, shopID, technicianID string) (*BusCapture, error) {
	var c BusCapture
	var modules, identity, live, coverage []byte
	err := s.pool.QueryRow(ctx, `
		SELECT vin, profile, adapter_type, host_os, protocol, make_hint, model_hint, year_hint,
		       modules, identity, live, active_codes, coverage, raw_hex_excerpt, scan_count,
		       first_seen_at, last_seen_at
		FROM bus_captures
		WHERE vin = $1
		  AND (
		        ($2 <> '' AND shop_id::text = $2)
		        OR ($2 = '' AND $3 <> '' AND shop_id IS NULL AND technician_id::text = $3)
		      )
	`, vin, strings.TrimSpace(shopID), strings.TrimSpace(technicianID)).Scan(
		&c.VIN, &c.Profile, &c.AdapterType, &c.HostOS, &c.Protocol, &c.MakeHint, &c.ModelHint, &c.YearHint,
		&modules, &identity, &live, &c.ActiveCodes, &coverage, &c.RawHexExcerpt, &c.ScanCount,
		&c.FirstSeenAt, &c.LastSeenAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(modules, &c.Modules)
	_ = json.Unmarshal(identity, &c.Identity)
	_ = json.Unmarshal(live, &c.Live)
	c.Coverage = coverage
	if c.Modules == nil {
		c.Modules = []BusModule{}
	}
	if c.Identity == nil {
		c.Identity = []BusIdentity{}
	}
	if c.Live == nil {
		c.Live = []BusLive{}
	}
	if c.ActiveCodes == nil {
		c.ActiveCodes = []string{}
	}
	return &c, nil
}

func (s *Store) UpsertBusCapture(ctx context.Context, vin, shopID, technicianID string, in BusCaptureIn) (BusCapture, error) {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	prev, err := s.BusCapture(ctx, vin, shopID, technicianID)
	if err != nil {
		return BusCapture{}, err
	}
	next := BusCapture{
		VIN:           vin,
		Profile:       strings.TrimSpace(in.Profile),
		AdapterType:   strings.TrimSpace(in.AdapterType),
		HostOS:        strings.TrimSpace(in.HostOS),
		Protocol:      strings.TrimSpace(in.Protocol),
		MakeHint:      strings.TrimSpace(in.MakeHint),
		ModelHint:     strings.TrimSpace(in.ModelHint),
		YearHint:      in.YearHint,
		Modules:       in.Modules,
		Identity:      in.Identity,
		Live:          in.Live,
		ActiveCodes:   in.ActiveCodes,
		Coverage:      in.Coverage,
		RawHexExcerpt: hexExcerpt(in.RawHexStream),
		ScanCount:     1,
		LastSeenAt:    time.Now(),
	}
	if next.ActiveCodes == nil {
		next.ActiveCodes = []string{}
	}
	if len(next.Coverage) == 0 {
		next.Coverage = json.RawMessage(`{}`)
	}
	if prev != nil {
		next.Modules = mergeBusModules(prev.Modules, in.Modules)
		next.Identity = mergeBusIdentity(prev.Identity, in.Identity)
		next.Live = mergeBusLive(prev.Live, in.Live)
		next.ScanCount = prev.ScanCount + 1
		next.FirstSeenAt = prev.FirstSeenAt
		if next.Profile == "" {
			next.Profile = prev.Profile
		}
		if next.MakeHint == "" {
			next.MakeHint = prev.MakeHint
		}
		if next.ModelHint == "" || strings.EqualFold(next.ModelHint, "unknown") {
			next.ModelHint = prev.ModelHint
		}
		if next.YearHint == 0 {
			next.YearHint = prev.YearHint
		}
		if next.RawHexExcerpt == "" {
			next.RawHexExcerpt = prev.RawHexExcerpt
		}
	} else {
		next.Modules = mergeBusModules(nil, in.Modules)
		next.FirstSeenAt = next.LastSeenAt
	}

	modJSON, err := json.Marshal(next.Modules)
	if err != nil {
		return BusCapture{}, err
	}
	idJSON, err := json.Marshal(next.Identity)
	if err != nil {
		return BusCapture{}, err
	}
	liveJSON, err := json.Marshal(next.Live)
	if err != nil {
		return BusCapture{}, err
	}
	var shop any
	if strings.TrimSpace(shopID) != "" {
		shop = shopID
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE bus_captures
		SET technician_id = $3::uuid,
		    profile = $4, adapter_type = $5, host_os = $6, protocol = $7,
		    make_hint = $8, model_hint = $9, year_hint = $10,
		    modules = $11, identity = $12, live = $13, active_codes = $14,
		    coverage = $15, raw_hex_excerpt = $16, scan_count = $17, last_seen_at = NOW()
		WHERE vin = $1
		  AND (
		        ($2 <> '' AND shop_id::text = $2)
		        OR ($2 = '' AND shop_id IS NULL AND technician_id::text = $3)
		      )
	`, vin, strings.TrimSpace(shopID), technicianID,
		next.Profile, next.AdapterType, next.HostOS, next.Protocol,
		next.MakeHint, next.ModelHint, next.YearHint,
		modJSON, idJSON, liveJSON, next.ActiveCodes,
		next.Coverage, next.RawHexExcerpt, next.ScanCount,
	)
	if err != nil {
		return BusCapture{}, err
	}
	if tag.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO bus_captures (
				shop_id, technician_id, vin, profile, adapter_type, host_os, protocol,
				make_hint, model_hint, year_hint, modules, identity, live, active_codes,
				coverage, raw_hex_excerpt, scan_count
			) VALUES (
				$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17
			)
		`, shop, technicianID, vin, next.Profile, next.AdapterType, next.HostOS, next.Protocol,
			next.MakeHint, next.ModelHint, next.YearHint, modJSON, idJSON, liveJSON, next.ActiveCodes,
			next.Coverage, next.RawHexExcerpt, next.ScanCount)
		if err != nil {
			return BusCapture{}, err
		}
	}
	saved, err := s.BusCapture(ctx, vin, shopID, technicianID)
	if err != nil {
		return BusCapture{}, err
	}
	if saved == nil {
		return next, nil
	}
	return *saved, nil
}

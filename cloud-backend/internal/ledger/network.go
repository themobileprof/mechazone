package ledger

import (
	"context"
	"strings"
	"time"
)

// NetworkMatch is this shop's similar-platform closeout. Never another shop's file on this VIN.
type NetworkMatch struct {
	ID         string    `json:"id"`
	DTC        string    `json:"diagnostic_trouble_code"`
	RootCause  string    `json:"root_cause_explanation"`
	Parts      []string  `json:"parts_replaced"`
	Verified   bool      `json:"is_verified_fix"`
	Reputation int       `json:"reputation_score"`
	Make       string    `json:"make"`
	Model      string    `json:"model"`
	Year       int       `json:"manufacture_year"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) SessionByID(ctx context.Context, id string) (Session, error) {
	var sess Session
	var codes []string
	var ff []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, vin, COALESCE(shop_id::text, ''), technician_id, mileage, adapter_type, host_os, protocol,
		       active_dtc_list, freeze_frame_telemetry, COALESCE(raw_hex_excerpt, ''), outcome, created_at
		FROM diagnostic_sessions WHERE id = $1
	`, id).Scan(
		&sess.ID, &sess.VIN, &sess.ShopID, &sess.TechnicianID, &sess.Mileage,
		&sess.AdapterType, &sess.HostOS, &sess.Protocol, &codes, &ff,
		&sess.RawHexExcerpt, &sess.Outcome, &sess.CreatedAt,
	)
	if err != nil {
		return Session{}, err
	}
	sess.ActiveCodes = codes
	if ff == nil {
		sess.FreezeFrame = []byte("null")
	} else {
		sess.FreezeFrame = ff
	}
	return sess, nil
}

func InShopScope(shopID, sessionShop, technicianID, sessionTech string) bool {
	shopID = strings.TrimSpace(shopID)
	sessionShop = strings.TrimSpace(sessionShop)
	technicianID = strings.TrimSpace(technicianID)
	sessionTech = strings.TrimSpace(sessionTech)
	if shopID != "" {
		return sessionShop == shopID
	}
	return technicianID != "" && sessionTech == technicianID && sessionShop == ""
}

func (s *Store) NetworkMatches(ctx context.Context, vin, shopID, technicianID, makeName, model string, year int, codes []string) ([]NetworkMatch, error) {
	makeName = strings.TrimSpace(makeName)
	model = strings.TrimSpace(model)
	shopID = strings.TrimSpace(shopID)
	technicianID = strings.TrimSpace(technicianID)
	if makeName == "" || model == "" {
		return []NetworkMatch{}, nil
	}
	lo, hi := 0, 9999
	if year > 0 {
		lo, hi = year-3, year+3
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.diagnostic_trouble_code, r.root_cause_explanation, r.parts_replaced,
		       r.is_verified_fix, t.reputation_score, v.make, v.model, v.manufacture_year, r.created_at
		FROM confirmed_resolutions r
		JOIN diagnostic_sessions s ON s.id = r.session_id
		JOIN vehicles v ON v.vin = r.vin
		JOIN technicians t ON t.id = r.technician_id
		WHERE r.vin <> $1
		  AND (
		        ($7 <> '' AND s.shop_id::text = $7)
		        OR ($7 = '' AND $8 <> '' AND s.shop_id IS NULL AND s.technician_id::text = $8)
		      )
		  AND lower(v.make) = lower($2)
		  AND lower(v.model) = lower($3)
		  AND v.manufacture_year BETWEEN $4 AND $5
		  AND (cardinality($6::text[]) = 0 OR r.diagnostic_trouble_code = ANY($6))
		ORDER BY r.is_verified_fix DESC, t.reputation_score DESC, r.created_at DESC
		LIMIT 20
	`, vin, makeName, model, lo, hi, codes, shopID, technicianID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NetworkMatch{}
	for rows.Next() {
		var m NetworkMatch
		if err := rows.Scan(
			&m.ID, &m.DTC, &m.RootCause, &m.Parts, &m.Verified,
			&m.Reputation, &m.Make, &m.Model, &m.Year, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) DTCTitles(ctx context.Context, codes []string) (map[string]DTC, error) {
	out := map[string]DTC{}
	for _, code := range codes {
		d, err := s.LookupDTC(ctx, code)
		if err != nil {
			return nil, err
		}
		out[d.Code] = d
	}
	return out, nil
}

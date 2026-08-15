package ledger

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mechazone/cloud-backend/internal/vin"
	"mechazone/cloud-backend/migrations"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, name := range names {
		var applied bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			return err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SeedDTCs(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open dtc seed: %w", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return err
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	batch := &pgx.Batch{}
	for {
		row, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		code := strings.ToUpper(strings.TrimSpace(row[idx["code"]]))
		category := row[idx["category"]]
		title := row[idx["title"]]
		source := row[idx["source"]]
		batch.Queue(
			`INSERT INTO dtc_codes (code, category, title, source) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (code) DO UPDATE SET title = EXCLUDED.title, category = EXCLUDED.category, source = EXCLUDED.source`,
			code, category, title, source,
		)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LookupDTC(ctx context.Context, code string) (DTC, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var d DTC
	err := s.pool.QueryRow(ctx, `SELECT code, category, title, source FROM dtc_codes WHERE code = $1`, code).
		Scan(&d.Code, &d.Category, &d.Title, &d.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return DTC{Code: code, Category: "unknown", Title: "", Source: ""}, nil
	}
	return d, err
}

func (s *Store) CachedVIN(ctx context.Context, vin string) (json.RawMessage, string, bool, error) {
	var payload []byte
	var source string
	err := s.pool.QueryRow(ctx, `SELECT payload, source FROM vin_decode_cache WHERE vin = $1`, vin).Scan(&payload, &source)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	return payload, source, true, nil
}

func (s *Store) SaveVINDecode(ctx context.Context, dec vin.Decode) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vin_decode_cache (vin, payload, source)
		VALUES ($1, $2, $3)
		ON CONFLICT (vin) DO NOTHING
	`, dec.VIN, dec.Raw, dec.Source)
	if err != nil {
		return err
	}
	makeName := dec.Make
	model := dec.Model
	year := dec.Year
	if makeName == "" {
		makeName = "Unknown"
	}
	if model == "" {
		model = "Unknown"
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO vehicles (vin, make, model, manufacture_year, decode_source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (vin) DO NOTHING
	`, dec.VIN, makeName, model, year, dec.Source)
	return err
}

func (s *Store) EnsureVehicle(ctx context.Context, vin, makeName, model string, year int, source string) error {
	if makeName == "" {
		makeName = "Unknown"
	}
	if model == "" {
		model = "Unknown"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vehicles (vin, make, model, manufacture_year, decode_source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (vin) DO NOTHING
	`, vin, makeName, model, year, source)
	return err
}

func (s *Store) History(ctx context.Context, vin string) (History, error) {
	h := History{FirstSeen: true, Sessions: []Session{}, Resolutions: []Resolution{}}
	var v Vehicle
	err := s.pool.QueryRow(ctx, `
		SELECT vin, make, model, manufacture_year, COALESCE(decode_source, ''), first_seen_at
		FROM vehicles WHERE vin = $1
	`, vin).Scan(&v.VIN, &v.Make, &v.Model, &v.Year, &v.DecodeSource, &v.FirstSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, nil
	}
	if err != nil {
		return History{}, err
	}
	h.Vehicle = &v
	h.FirstSeen = false

	rows, err := s.pool.Query(ctx, `
		SELECT id, vin, COALESCE(shop_id::text, ''), technician_id, mileage, adapter_type, host_os, protocol,
		       active_dtc_list, freeze_frame_telemetry, COALESCE(raw_hex_excerpt, ''), outcome, created_at
		FROM diagnostic_sessions
		WHERE vin = $1
		ORDER BY created_at DESC
	`, vin)
	if err != nil {
		return History{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var sess Session
		var codes []string
		var ff []byte
		if err := rows.Scan(
			&sess.ID, &sess.VIN, &sess.ShopID, &sess.TechnicianID, &sess.Mileage,
			&sess.AdapterType, &sess.HostOS, &sess.Protocol, &codes, &ff,
			&sess.RawHexExcerpt, &sess.Outcome, &sess.CreatedAt,
		); err != nil {
			return History{}, err
		}
		sess.ActiveCodes = codes
		if ff == nil {
			sess.FreezeFrame = json.RawMessage("null")
		} else {
			sess.FreezeFrame = ff
		}
		h.Sessions = append(h.Sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return History{}, err
	}

	rrows, err := s.pool.Query(ctx, `
		SELECT id, session_id, vin, technician_id, diagnostic_trouble_code,
		       root_cause_explanation, parts_replaced, is_verified_fix, created_at
		FROM confirmed_resolutions
		WHERE vin = $1
		ORDER BY created_at DESC
	`, vin)
	if err != nil {
		return History{}, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var res Resolution
		if err := rrows.Scan(
			&res.ID, &res.SessionID, &res.VIN, &res.TechnicianID, &res.DTC,
			&res.RootCause, &res.PartsReplaced, &res.Verified, &res.CreatedAt,
		); err != nil {
			return History{}, err
		}
		h.Resolutions = append(h.Resolutions, res)
	}
	return h, rrows.Err()
}

func (s *Store) InsertSession(ctx context.Context, in SessionIngest) (Session, error) {
	excerpt := strings.Join(in.RawHexStream, "\n")
	if len(excerpt) > 8000 {
		excerpt = excerpt[:8000]
	}
	ff := in.FreezeFrame
	if len(ff) == 0 {
		ff = json.RawMessage("null")
	}
	var shop any
	if strings.TrimSpace(in.ShopID) != "" {
		shop = in.ShopID
	}
	var sess Session
	var codes []string
	var rawFF []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO diagnostic_sessions (
			vin, shop_id, technician_id, mileage, adapter_type, host_os, protocol,
			active_dtc_list, freeze_frame_telemetry, raw_hex_excerpt, outcome
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'open')
		RETURNING id, vin, COALESCE(shop_id::text, ''), technician_id, mileage, adapter_type, host_os, protocol,
		          active_dtc_list, freeze_frame_telemetry, COALESCE(raw_hex_excerpt, ''), outcome, created_at
	`, in.VIN, shop, in.TechnicianID, in.MileageKM, in.AdapterType, in.HostOS, in.Protocol,
		in.ActiveCodes, ff, excerpt,
	).Scan(
		&sess.ID, &sess.VIN, &sess.ShopID, &sess.TechnicianID, &sess.Mileage,
		&sess.AdapterType, &sess.HostOS, &sess.Protocol, &codes, &rawFF,
		&sess.RawHexExcerpt, &sess.Outcome, &sess.CreatedAt,
	)
	if err != nil {
		return Session{}, err
	}
	sess.ActiveCodes = codes
	sess.FreezeFrame = rawFF
	return sess, nil
}

func (s *Store) Closeout(ctx context.Context, sessionID, technicianID string, c Closeout) (Resolution, error) {
	outcome := strings.ToLower(strings.TrimSpace(c.Outcome))
	if outcome != "success" && outcome != "failed" {
		return Resolution{}, fmt.Errorf("outcome must be success or failed")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Resolution{}, err
	}
	defer tx.Rollback(ctx)

	var vin, tech string
	err = tx.QueryRow(ctx, `SELECT vin, technician_id FROM diagnostic_sessions WHERE id = $1`, sessionID).
		Scan(&vin, &tech)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resolution{}, fmt.Errorf("session not found")
	}
	if err != nil {
		return Resolution{}, err
	}
	if technicianID == "" || tech != technicianID {
		return Resolution{}, fmt.Errorf("only the technician who opened this session can close it")
	}
	if _, err := tx.Exec(ctx, `UPDATE diagnostic_sessions SET outcome = $2 WHERE id = $1`, sessionID, outcome); err != nil {
		return Resolution{}, err
	}
	verified := outcome == "success"
	var res Resolution
	err = tx.QueryRow(ctx, `
		INSERT INTO confirmed_resolutions (
			session_id, vin, technician_id, diagnostic_trouble_code,
			root_cause_explanation, parts_replaced, is_verified_fix
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, session_id, vin, technician_id, diagnostic_trouble_code,
		          root_cause_explanation, parts_replaced, is_verified_fix, created_at
	`, sessionID, vin, technicianID, strings.ToUpper(c.DTC), c.RootCause, c.Parts, verified,
	).Scan(
		&res.ID, &res.SessionID, &res.VIN, &res.TechnicianID, &res.DTC,
		&res.RootCause, &res.PartsReplaced, &res.Verified, &res.CreatedAt,
	)
	if err != nil {
		return Resolution{}, err
	}
	if verified {
		if _, err := tx.Exec(ctx, `
			UPDATE technicians SET reputation_score = reputation_score + 5 WHERE id = $1
		`, technicianID); err != nil {
			return Resolution{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Resolution{}, err
	}
	return res, nil
}

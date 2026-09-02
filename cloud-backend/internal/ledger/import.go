package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"mechazone/cloud-backend/internal/importreport"
)

type StoredImport struct {
	JobImport
	StoragePath string
}

func (s *Store) InsertImportedReport(ctx context.Context, in SessionIngest, meta JobImport, data []byte, root, ext string) (Session, JobImport, error) {
	if strings.TrimSpace(root) == "" {
		return Session{}, JobImport{}, fmt.Errorf("import dir is not configured")
	}
	if len(data) == 0 {
		return Session{}, JobImport{}, fmt.Errorf("empty file")
	}
	if len(data) > importreport.MaxBytes {
		return Session{}, JobImport{}, fmt.Errorf("file exceeds 8 MB")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, JobImport{}, err
	}
	defer tx.Rollback(ctx)

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
	err = tx.QueryRow(ctx, `
		INSERT INTO diagnostic_sessions (
			vin, shop_id, technician_id, mileage, adapter_type, host_os, protocol,
			active_dtc_list, freeze_frame_telemetry, raw_hex_excerpt, outcome
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'open')
		RETURNING id, vin, COALESCE(shop_id::text, ''), technician_id, mileage, adapter_type, host_os, protocol,
		          active_dtc_list, freeze_frame_telemetry, COALESCE(raw_hex_excerpt, ''), outcome, created_at
	`, in.VIN, shop, in.TechnicianID, in.MileageKM, in.AdapterType, in.HostOS, in.Protocol,
		in.ActiveCodes, ff, "",
	).Scan(
		&sess.ID, &sess.VIN, &sess.ShopID, &sess.TechnicianID, &sess.Mileage,
		&sess.AdapterType, &sess.HostOS, &sess.Protocol, &codes, &rawFF,
		&sess.RawHexExcerpt, &sess.Outcome, &sess.CreatedAt,
	)
	if err != nil {
		return Session{}, JobImport{}, err
	}
	sess.ActiveCodes = codes
	sess.FreezeFrame = rawFF

	rel := filepath.Join(importreport.ScopeDir(in.ShopID, in.TechnicianID), sess.ID+ext)
	full, err := importreport.ResolveStorage(root, rel)
	if err != nil {
		return Session{}, JobImport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return Session{}, JobImport{}, err
	}
	if err := os.WriteFile(full, data, 0o600); err != nil {
		return Session{}, JobImport{}, err
	}
	wrote := true
	defer func() {
		if wrote {
			_ = os.Remove(full)
		}
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO session_imports (session_id, source, original_name, content_type, byte_size, note, storage_path)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, sess.ID, meta.Source, meta.OriginalName, meta.ContentType, len(data), meta.Note, rel)
	if err != nil {
		return Session{}, JobImport{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, JobImport{}, err
	}
	wrote = false
	meta.ByteSize = len(data)
	return sess, meta, nil
}

func (s *Store) ImportBySession(ctx context.Context, sessionID string) (StoredImport, error) {
	var row StoredImport
	err := s.pool.QueryRow(ctx, `
		SELECT source, original_name, content_type, byte_size, note, storage_path
		FROM session_imports WHERE session_id = $1
	`, sessionID).Scan(&row.Source, &row.OriginalName, &row.ContentType, &row.ByteSize, &row.Note, &row.StoragePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredImport{}, fmt.Errorf("import not found")
	}
	if err != nil {
		return StoredImport{}, err
	}
	return row, nil
}

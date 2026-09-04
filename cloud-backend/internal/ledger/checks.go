package ledger

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	CheckOpen     = "open"
	CheckDone     = "done"
	CheckRuledOut = "ruled_out"
)

// PlaybookCheck is one playbook step this shop ticked on a VIN. Not a job closeout.
type PlaybookCheck struct {
	ID          string    `json:"id"`
	VIN         string    `json:"vin"`
	Fingerprint string    `json:"fingerprint"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Detail      string    `json:"detail"`
	Status      string    `json:"status"`
	Note        string    `json:"note,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaybookCheckIn is the bay tick. Shop and technician IDs come from the cookie.
type PlaybookCheckIn struct {
	Fingerprint string `json:"fingerprint"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Status      string `json:"status"`
	Note        string `json:"note"`
}

// PlaybookStepSeed is a playbook step used to open a check without importing the AI package.
type PlaybookStepSeed struct {
	Kind   string
	Title  string
	Detail string
}

func CheckFingerprint(kind, title string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	t := strings.ToLower(strings.TrimSpace(title))
	if k == "" {
		k = "test"
	}
	return k + "|" + t
}

func normalizeCheckStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case CheckDone:
		return CheckDone
	case CheckRuledOut, "not_this", "ruled-out":
		return CheckRuledOut
	default:
		return CheckOpen
	}
}

func scopeSQL() string {
	return `
		  AND (
		        ($2 <> '' AND shop_id::text = $2)
		        OR ($2 = '' AND $3 <> '' AND shop_id IS NULL AND technician_id::text = $3)
		      )`
}

func (s *Store) PlaybookChecks(ctx context.Context, vin, shopID, technicianID string) ([]PlaybookCheck, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, vin, fingerprint, kind, title, detail, status, note, updated_at
		FROM playbook_checks
		WHERE vin = $1
		`+scopeSQL()+`
		ORDER BY updated_at DESC, title
	`, vin, strings.TrimSpace(shopID), strings.TrimSpace(technicianID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlaybookCheck{}
	for rows.Next() {
		var c PlaybookCheck
		if err := rows.Scan(&c.ID, &c.VIN, &c.Fingerprint, &c.Kind, &c.Title, &c.Detail, &c.Status, &c.Note, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) UpsertPlaybookCheck(ctx context.Context, vin, shopID, technicianID string, in PlaybookCheckIn) (PlaybookCheck, error) {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.Kind == "" {
		in.Kind = "test"
	}
	in.Title = clipRunes(strings.TrimSpace(in.Title), 200)
	in.Detail = clipRunes(strings.TrimSpace(in.Detail), 800)
	in.Note = clipRunes(strings.TrimSpace(in.Note), 500)
	in.Status = normalizeCheckStatus(in.Status)
	fp := strings.TrimSpace(in.Fingerprint)
	if fp == "" {
		fp = CheckFingerprint(in.Kind, in.Title)
	}
	if in.Title == "" || fp == "" {
		return PlaybookCheck{}, errors.New("check title is required")
	}
	var shop any
	if strings.TrimSpace(shopID) != "" {
		shop = shopID
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE playbook_checks
		SET technician_id = $3::uuid,
		    kind = $5, title = $6, detail = $7, status = $8, note = $9, updated_at = NOW()
		WHERE vin = $1
		  AND fingerprint = $4
		`+scopeSQL()+`
	`, vin, strings.TrimSpace(shopID), technicianID, fp, in.Kind, in.Title, in.Detail, in.Status, in.Note)
	if err != nil {
		return PlaybookCheck{}, err
	}
	if tag.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO playbook_checks (
				shop_id, technician_id, vin, fingerprint, kind, title, detail, status, note
			) VALUES (
				$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9
			)
		`, shop, technicianID, vin, fp, in.Kind, in.Title, in.Detail, in.Status, in.Note)
		if err != nil {
			return PlaybookCheck{}, err
		}
	}
	var c PlaybookCheck
	err = s.pool.QueryRow(ctx, `
		SELECT id::text, vin, fingerprint, kind, title, detail, status, note, updated_at
		FROM playbook_checks
		WHERE vin = $1 AND fingerprint = $4
		`+scopeSQL()+`
	`, vin, strings.TrimSpace(shopID), technicianID, fp).Scan(
		&c.ID, &c.VIN, &c.Fingerprint, &c.Kind, &c.Title, &c.Detail, &c.Status, &c.Note, &c.UpdatedAt,
	)
	if err != nil {
		return PlaybookCheck{}, err
	}
	return c, nil
}

// SyncPlaybookSteps opens a check for each new playbook step. Settled ticks are kept.
func (s *Store) SyncPlaybookSteps(ctx context.Context, vin, shopID, technicianID string, steps []PlaybookStepSeed) ([]PlaybookCheck, error) {
	for _, st := range steps {
		title := strings.TrimSpace(st.Title)
		if title == "" {
			continue
		}
		fp := CheckFingerprint(st.Kind, title)
		existing, err := s.checkByFingerprint(ctx, vin, shopID, technicianID, fp)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status != CheckOpen {
			continue
		}
		status := CheckOpen
		note := ""
		if existing != nil {
			status = existing.Status
			note = existing.Note
		}
		if _, err := s.UpsertPlaybookCheck(ctx, vin, shopID, technicianID, PlaybookCheckIn{
			Fingerprint: fp,
			Kind:        st.Kind,
			Title:       title,
			Detail:      st.Detail,
			Status:      status,
			Note:        note,
		}); err != nil {
			return nil, err
		}
	}
	return s.PlaybookChecks(ctx, vin, shopID, technicianID)
}

func (s *Store) checkByFingerprint(ctx context.Context, vin, shopID, technicianID, fp string) (*PlaybookCheck, error) {
	var c PlaybookCheck
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, vin, fingerprint, kind, title, detail, status, note, updated_at
		FROM playbook_checks
		WHERE vin = $1 AND fingerprint = $4
		`+scopeSQL()+`
	`, vin, strings.TrimSpace(shopID), technicianID, fp).Scan(
		&c.ID, &c.VIN, &c.Fingerprint, &c.Kind, &c.Title, &c.Detail, &c.Status, &c.Note, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

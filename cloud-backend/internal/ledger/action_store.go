package ledger

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// PlaybookAction is one morphed cluster of AI playbook steps. Not a VIN job.
type PlaybookAction struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Tokens      []string  `json:"tokens"`
	Variants    []string  `json:"variants"`
	SeenCount   int       `json:"seen_count"`
	GuideIDs    []string  `json:"guide_ids,omitempty"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// ObservedStep is the catalog row plus how-to cards that apply to that AI step.
type ObservedStep struct {
	Action   PlaybookAction
	GuideIDs []string
}

func (s *Store) ListPlaybookActions(ctx context.Context) ([]PlaybookAction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id::text, a.fingerprint, a.kind, a.title, a.tokens, a.variants, a.seen_count,
		       a.first_seen_at, a.last_seen_at,
		       coalesce(array_agg(g.guide_id::text) FILTER (WHERE g.guide_id IS NOT NULL), '{}')
		FROM playbook_actions a
		LEFT JOIN howto_guide_actions g ON g.action_id = a.id
		GROUP BY a.id
		ORDER BY a.seen_count DESC, a.last_seen_at DESC, a.title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlaybookAction{}
	for rows.Next() {
		var a PlaybookAction
		if err := rows.Scan(&a.ID, &a.Fingerprint, &a.Kind, &a.Title, &a.Tokens, &a.Variants, &a.SeenCount,
			&a.FirstSeenAt, &a.LastSeenAt, &a.GuideIDs); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) actionsByKind(ctx context.Context, kind string) ([]PlaybookAction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, fingerprint, kind, title, tokens, variants, seen_count, first_seen_at, last_seen_at
		FROM playbook_actions
		WHERE kind = $1
	`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlaybookAction{}
	for rows.Next() {
		var a PlaybookAction
		if err := rows.Scan(&a.ID, &a.Fingerprint, &a.Kind, &a.Title, &a.Tokens, &a.Variants, &a.SeenCount, &a.FirstSeenAt, &a.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func bestMerge(existing []PlaybookAction, kind string, tokens []string) *PlaybookAction {
	fp := ActionFingerprint(kind, tokens)
	var best *PlaybookAction
	bestJ := -1.0
	for i := range existing {
		row := &existing[i]
		if row.Fingerprint == fp {
			return row
		}
		if !ShouldMerge(kind, row.Kind, tokens, row.Tokens) {
			continue
		}
		j := Jaccard(tokens, row.Tokens)
		if j > bestJ {
			bestJ = j
			best = row
		}
	}
	return best
}

func (s *Store) bumpAction(ctx context.Context, id, title string, tokens []string) (PlaybookAction, error) {
	var a PlaybookAction
	err := s.pool.QueryRow(ctx, `
		UPDATE playbook_actions
		SET seen_count = seen_count + 1,
		    last_seen_at = NOW(),
		    variants = CASE
		        WHEN $2 = title THEN variants
		        WHEN $2 = ANY (variants) THEN variants
		        WHEN coalesce(array_length(variants, 1), 0) >= 16 THEN variants
		        ELSE array_append(variants, $2)
		    END
		WHERE id = $1::uuid
		RETURNING id::text, fingerprint, kind, title, tokens, variants, seen_count, first_seen_at, last_seen_at
	`, id, clip(strings.TrimSpace(title), 200)).Scan(
		&a.ID, &a.Fingerprint, &a.Kind, &a.Title, &a.Tokens, &a.Variants, &a.SeenCount, &a.FirstSeenAt, &a.LastSeenAt)
	if err != nil {
		return PlaybookAction{}, err
	}
	_ = tokens
	return a, nil
}

func (s *Store) insertAction(ctx context.Context, fp, kind, title string, tokens []string) (PlaybookAction, error) {
	var a PlaybookAction
	err := s.pool.QueryRow(ctx, `
		INSERT INTO playbook_actions (fingerprint, kind, title, tokens)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, fingerprint, kind, title, tokens, variants, seen_count, first_seen_at, last_seen_at
	`, fp, kind, clip(strings.TrimSpace(title), 200), tokens).Scan(
		&a.ID, &a.Fingerprint, &a.Kind, &a.Title, &a.Tokens, &a.Variants, &a.SeenCount, &a.FirstSeenAt, &a.LastSeenAt)
	if err != nil {
		return PlaybookAction{}, err
	}
	return a, nil
}

func (s *Store) ObserveAction(ctx context.Context, kind, title string) (PlaybookAction, error) {
	kind = normalizeActionKind(kind)
	title = strings.TrimSpace(title)
	if title == "" {
		return PlaybookAction{}, errors.New("action title required")
	}
	tokens := ActionTokens(kind, title)
	fp := ActionFingerprint(kind, tokens)
	existing, err := s.actionsByKind(ctx, kind)
	if err != nil {
		return PlaybookAction{}, err
	}
	if hit := bestMerge(existing, kind, tokens); hit != nil {
		return s.bumpAction(ctx, hit.ID, title, tokens)
	}
	a, err := s.insertAction(ctx, fp, kind, title, tokens)
	if err != nil {
		if isUniqueViolation(err) {
			existing, err = s.actionsByKind(ctx, kind)
			if err != nil {
				return PlaybookAction{}, err
			}
			if hit := bestMerge(existing, kind, tokens); hit != nil {
				return s.bumpAction(ctx, hit.ID, title, tokens)
			}
		}
		return PlaybookAction{}, err
	}
	return a, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}

func (s *Store) ObservePlaybookSteps(ctx context.Context, seeds []PlaybookStepSeed) ([]ObservedStep, error) {
	out := make([]ObservedStep, 0, len(seeds))
	for _, seed := range seeds {
		a, err := s.ObserveAction(ctx, seed.Kind, seed.Title)
		if err != nil {
			return nil, err
		}
		ids, err := s.guideIDsForStep(ctx, a.ID, seed.Kind, seed.Title, seed.Detail)
		if err != nil {
			return nil, err
		}
		a.GuideIDs = ids
		out = append(out, ObservedStep{Action: a, GuideIDs: ids})
	}
	return out, nil
}

func (s *Store) guideIDsForStep(ctx context.Context, actionID, kind, title, detail string) ([]string, error) {
	blob := strings.ToLower(strings.TrimSpace(kind + " " + title + " " + detail))
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT g.id::text
		FROM howto_guides g
		LEFT JOIN howto_guide_actions a ON a.guide_id = g.id
		WHERE g.published
		  AND (
		        a.action_id = $1::uuid
		        OR EXISTS (
		            SELECT 1 FROM unnest(g.match_words) w
		            WHERE w <> '' AND position(lower(w) IN $2) > 0
		        )
		      )
		ORDER BY 1
	`, actionID, blob)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) actionByID(ctx context.Context, id string) (PlaybookAction, error) {
	var a PlaybookAction
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, fingerprint, kind, title, tokens, variants, seen_count, first_seen_at, last_seen_at
		FROM playbook_actions WHERE id = $1::uuid
	`, id).Scan(&a.ID, &a.Fingerprint, &a.Kind, &a.Title, &a.Tokens, &a.Variants, &a.SeenCount, &a.FirstSeenAt, &a.LastSeenAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PlaybookAction{}, errors.New("action not found")
		}
		return PlaybookAction{}, err
	}
	return a, nil
}

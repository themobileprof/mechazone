package ledger

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// HowToGuide is an admin-authored bay card. Generic shop skill, not this VIN.
type HowToGuide struct {
	ID         string    `json:"id"`
	Slug       string    `json:"slug,omitempty"`
	Title      string    `json:"title"`
	Blurb      string    `json:"blurb"`
	Warning    string    `json:"warning"`
	BodyHTML   string    `json:"body_html"`
	MatchWords []string  `json:"match_words"`
	Published  bool      `json:"published"`
	ActionIDs  []string  `json:"action_ids,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// HowToIn is create/update from super admin.
type HowToIn struct {
	Title      string   `json:"title"`
	Blurb      string   `json:"blurb"`
	Warning    string   `json:"warning"`
	BodyHTML   string   `json:"body_html"`
	MatchWords []string `json:"match_words"`
	Published  *bool    `json:"published"`
	ActionIDs  []string `json:"action_ids"`
}

type howtoScanner interface {
	Scan(dest ...any) error
}

func scanHowTo(row howtoScanner) (HowToGuide, error) {
	var g HowToGuide
	var slug *string
	err := row.Scan(&g.ID, &slug, &g.Title, &g.Blurb, &g.Warning, &g.BodyHTML, &g.MatchWords, &g.Published, &g.CreatedAt, &g.UpdatedAt, &g.ActionIDs)
	if slug != nil {
		g.Slug = *slug
	}
	return g, err
}

const howToSelect = `
		SELECT g.id::text, g.slug, g.title, g.blurb, g.warning, g.body_html, g.match_words, g.published,
		       g.created_at, g.updated_at,
		       coalesce(array_agg(a.action_id::text) FILTER (WHERE a.action_id IS NOT NULL), '{}')
		FROM howto_guides g
		LEFT JOIN howto_guide_actions a ON a.guide_id = g.id`

func (s *Store) ListHowToGuides(ctx context.Context, publishedOnly bool) ([]HowToGuide, error) {
	q := howToSelect + ` WHERE ($1 = false OR g.published) GROUP BY g.id ORDER BY g.title`
	rows, err := s.pool.Query(ctx, q, publishedOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HowToGuide{}
	for rows.Next() {
		g, err := scanHowTo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GetHowToGuide(ctx context.Context, id string) (HowToGuide, error) {
	g, err := scanHowTo(s.pool.QueryRow(ctx, howToSelect+` WHERE g.id = $1::uuid GROUP BY g.id`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return HowToGuide{}, errors.New("how-to not found")
		}
		return HowToGuide{}, err
	}
	return g, nil
}

func (s *Store) GuidesByIDs(ctx context.Context, ids []string) ([]HowToGuide, error) {
	if len(ids) == 0 {
		return []HowToGuide{}, nil
	}
	rows, err := s.pool.Query(ctx, howToSelect+` WHERE g.id = ANY ($1::uuid[]) AND g.published GROUP BY g.id ORDER BY g.title`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HowToGuide{}
	for rows.Next() {
		g, err := scanHowTo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func publishedOr(p *bool, fallback bool) bool {
	if p == nil {
		return fallback
	}
	return *p
}

func (s *Store) CreateHowToGuide(ctx context.Context, in HowToIn) (HowToGuide, error) {
	title := clip(strings.TrimSpace(in.Title), 160)
	if title == "" {
		return HowToGuide{}, errors.New("title required")
	}
	body := SanitizeHowToHTML(in.BodyHTML)
	if body == "" {
		return HowToGuide{}, errors.New("how-to body required")
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO howto_guides (title, blurb, warning, body_html, match_words, published)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, title, clip(strings.TrimSpace(in.Blurb), 500), clip(strings.TrimSpace(in.Warning), 500),
		body, normalizeMatchWords(in.MatchWords), publishedOr(in.Published, true)).Scan(&id)
	if err != nil {
		return HowToGuide{}, err
	}
	if err := s.replaceGuideActions(ctx, id, in.ActionIDs); err != nil {
		return HowToGuide{}, err
	}
	return s.GetHowToGuide(ctx, id)
}

func (s *Store) UpdateHowToGuide(ctx context.Context, id string, in HowToIn) (HowToGuide, error) {
	title := clip(strings.TrimSpace(in.Title), 160)
	if title == "" {
		return HowToGuide{}, errors.New("title required")
	}
	body := SanitizeHowToHTML(in.BodyHTML)
	if body == "" {
		return HowToGuide{}, errors.New("how-to body required")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE howto_guides
		SET title = $2, blurb = $3, warning = $4, body_html = $5, match_words = $6, published = $7, updated_at = NOW()
		WHERE id = $1::uuid
	`, id, title, clip(strings.TrimSpace(in.Blurb), 500), clip(strings.TrimSpace(in.Warning), 500),
		body, normalizeMatchWords(in.MatchWords), publishedOr(in.Published, true))
	if err != nil {
		return HowToGuide{}, err
	}
	if tag.RowsAffected() == 0 {
		return HowToGuide{}, errors.New("how-to not found")
	}
	if err := s.replaceGuideActions(ctx, id, in.ActionIDs); err != nil {
		return HowToGuide{}, err
	}
	return s.GetHowToGuide(ctx, id)
}

func (s *Store) DeleteHowToGuide(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM howto_guides WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("how-to not found")
	}
	return nil
}

func (s *Store) replaceGuideActions(ctx context.Context, guideID string, actionIDs []string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM howto_guide_actions WHERE guide_id = $1::uuid`, guideID); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, aid := range actionIDs {
		aid = strings.TrimSpace(aid)
		if aid == "" {
			continue
		}
		if _, ok := seen[aid]; ok {
			continue
		}
		seen[aid] = struct{}{}
		if _, err := s.actionByID(ctx, aid); err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO howto_guide_actions (guide_id, action_id) VALUES ($1::uuid, $2::uuid)
		`, guideID, aid); err != nil {
			return err
		}
	}
	return nil
}

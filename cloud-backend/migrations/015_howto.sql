-- Canonical AI playbook actions (similar titles morph together) and admin how-to cards.
CREATE TABLE playbook_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL DEFAULT 'test',
    title TEXT NOT NULL,
    tokens TEXT[] NOT NULL DEFAULT '{}',
    variants TEXT[] NOT NULL DEFAULT '{}',
    seen_count INT NOT NULL DEFAULT 1,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX playbook_actions_kind ON playbook_actions (kind);
CREATE INDEX playbook_actions_seen ON playbook_actions (seen_count DESC, last_seen_at DESC);

CREATE TABLE howto_guides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE,
    title TEXT NOT NULL,
    blurb TEXT NOT NULL DEFAULT '',
    warning TEXT NOT NULL DEFAULT '',
    body_html TEXT NOT NULL DEFAULT '',
    match_words TEXT[] NOT NULL DEFAULT '{}',
    published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE howto_guide_actions (
    guide_id UUID NOT NULL REFERENCES howto_guides(id) ON DELETE CASCADE,
    action_id UUID NOT NULL REFERENCES playbook_actions(id) ON DELETE CASCADE,
    PRIMARY KEY (guide_id, action_id)
);

CREATE INDEX howto_guide_actions_action ON howto_guide_actions (action_id);

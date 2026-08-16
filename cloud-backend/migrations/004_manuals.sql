CREATE TABLE doc_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sha256 CHAR(64) NOT NULL UNIQUE,
    path TEXT NOT NULL,
    title TEXT NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'workshop_manual',
    make VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    year_from INT NOT NULL,
    year_to INT NOT NULL,
    engine VARCHAR(64) NOT NULL DEFAULT '',
    language VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE doc_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES doc_sources(id) ON DELETE CASCADE,
    page INT NOT NULL,
    language VARCHAR(16) NOT NULL,
    body TEXT NOT NULL,
    body_en TEXT NOT NULL DEFAULT '',
    codes TEXT[] NOT NULL DEFAULT '{}',
    tsv tsvector,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE doc_figures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES doc_sources(id) ON DELETE CASCADE,
    page INT NOT NULL,
    caption TEXT NOT NULL DEFAULT '',
    language VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_doc_sources_platform ON doc_sources (lower(make), lower(model), year_from, year_to);
CREATE INDEX idx_doc_chunks_source ON doc_chunks (source_id, page);
CREATE INDEX idx_doc_chunks_codes ON doc_chunks USING GIN (codes);
CREATE INDEX idx_doc_chunks_tsv ON doc_chunks USING GIN (tsv);
CREATE INDEX idx_doc_figures_source ON doc_figures (source_id, page);

CREATE OR REPLACE FUNCTION doc_chunks_tsv_refresh() RETURNS trigger AS $$
BEGIN
    NEW.tsv := to_tsvector('simple', coalesce(NEW.body, '') || ' ' || coalesce(NEW.body_en, '') || ' ' || coalesce(array_to_string(NEW.codes, ' '), ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER doc_chunks_tsv BEFORE INSERT OR UPDATE OF body, body_en, codes
    ON doc_chunks FOR EACH ROW EXECUTE FUNCTION doc_chunks_tsv_refresh();

-- One embedding model per ledger. Mixing bge-small and all-MiniLM (both 384-d) makes cosine garbage.
CREATE TABLE IF NOT EXISTS doc_embedding_meta (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    model TEXT NOT NULL,
    dim INT NOT NULL CHECK (dim > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'doc_chunks' AND column_name = 'embedding'
    ) THEN
        INSERT INTO doc_embedding_meta (singleton, model, dim)
        SELECT TRUE, 'bge-small-en-v1.5', 384
        WHERE EXISTS (SELECT 1 FROM doc_chunks WHERE embedding IS NOT NULL)
        ON CONFLICT (singleton) DO NOTHING;
    END IF;
END $$;

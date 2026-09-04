-- Resize doc_chunks.embedding to 384 (bge-small-en-v1.5). Drops any 1536 OpenAI vectors.
CREATE EXTENSION IF NOT EXISTS vector;

DO $$
DECLARE
  t text;
BEGIN
  SELECT format_type(a.atttypid, a.atttypmod) INTO t
  FROM pg_attribute a
  JOIN pg_class c ON c.oid = a.attrelid
  JOIN pg_namespace n ON n.oid = c.relnamespace
  WHERE n.nspname = 'public' AND c.relname = 'doc_chunks'
    AND a.attname = 'embedding' AND NOT a.attisdropped;
  IF t IS DISTINCT FROM 'vector(384)' THEN
    DROP INDEX IF EXISTS doc_chunks_embedding_hnsw;
    ALTER TABLE doc_chunks DROP COLUMN IF EXISTS embedding;
    ALTER TABLE doc_chunks ADD COLUMN embedding vector(384);
    CREATE INDEX doc_chunks_embedding_hnsw
      ON doc_chunks USING hnsw (embedding vector_cosine_ops)
      WHERE embedding IS NOT NULL;
  END IF;
END $$;

-- Hybrid RAG: cosine on ingested chunks. FTS (tsv) and DTC GIN stay.
-- Local default is BAAI/bge-small-en-v1.5 (384-dim) via Ollama. Requires pgvector.
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE doc_chunks
    ADD COLUMN IF NOT EXISTS embedding vector(384);

CREATE INDEX IF NOT EXISTS doc_chunks_embedding_hnsw
    ON doc_chunks USING hnsw (embedding vector_cosine_ops)
    WHERE embedding IS NOT NULL;

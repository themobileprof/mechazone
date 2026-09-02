-- Scanner PDFs/photos attached to a VIN. Not a live OpenPort session.
CREATE TABLE session_imports (
    session_id UUID PRIMARY KEY REFERENCES diagnostic_sessions(id) ON DELETE CASCADE,
    source VARCHAR(32) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(128) NOT NULL,
    byte_size INT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

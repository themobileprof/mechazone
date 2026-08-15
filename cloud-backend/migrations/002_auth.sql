ALTER TABLE technicians
    ALTER COLUMN shop_id DROP NOT NULL;

ALTER TABLE diagnostic_sessions
    ALTER COLUMN shop_id DROP NOT NULL;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(32) NOT NULL CHECK (role IN ('super_admin', 'technician')),
    technician_id UUID REFERENCES technicians(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT users_role_link CHECK (
        (role = 'super_admin' AND technician_id IS NULL)
        OR (role = 'technician' AND technician_id IS NOT NULL)
    )
);

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_auth_sessions_user ON auth_sessions (user_id);
CREATE INDEX idx_auth_sessions_expires ON auth_sessions (expires_at);
CREATE INDEX idx_users_email ON users (email);

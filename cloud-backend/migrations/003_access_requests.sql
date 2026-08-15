CREATE TABLE access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    applicant_name VARCHAR(120) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(32) NOT NULL DEFAULT '',
    shop_name VARCHAR(160) NOT NULL DEFAULT '',
    city VARCHAR(80) NOT NULL,
    country VARCHAR(80) NOT NULL DEFAULT 'Nigeria',
    kind VARCHAR(32) NOT NULL CHECK (kind IN ('shop', 'freelancer')),
    note TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'provisioned', 'dismissed')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_access_requests_status ON access_requests (status, created_at DESC);
CREATE INDEX idx_access_requests_email ON access_requests (contact_email);

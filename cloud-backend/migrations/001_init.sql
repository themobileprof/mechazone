CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location_country VARCHAR(100) DEFAULT 'Nigeria',
    location_city VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE technicians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID REFERENCES shops(id) ON DELETE RESTRICT,
    full_name VARCHAR(255) NOT NULL,
    reputation_score INT DEFAULT 100,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vehicles (
    vin VARCHAR(17) PRIMARY KEY,
    make VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    manufacture_year INT NOT NULL,
    decode_source VARCHAR(50),
    first_seen_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE diagnostic_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vin VARCHAR(17) REFERENCES vehicles(vin) ON DELETE CASCADE,
    shop_id UUID REFERENCES shops(id) ON DELETE RESTRICT,
    technician_id UUID REFERENCES technicians(id) ON DELETE RESTRICT,
    mileage INT NOT NULL,
    adapter_type VARCHAR(64) NOT NULL,
    host_os VARCHAR(32) NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    active_dtc_list TEXT[] NOT NULL,
    freeze_frame_telemetry JSONB,
    raw_hex_excerpt TEXT,
    outcome VARCHAR(16) DEFAULT 'open',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE confirmed_resolutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES diagnostic_sessions(id) ON DELETE CASCADE,
    vin VARCHAR(17) REFERENCES vehicles(vin) ON DELETE CASCADE,
    technician_id UUID REFERENCES technicians(id) ON DELETE RESTRICT,
    diagnostic_trouble_code VARCHAR(10) NOT NULL,
    root_cause_explanation TEXT NOT NULL,
    parts_replaced TEXT[] NOT NULL,
    is_verified_fix BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vin_decode_cache (
    vin VARCHAR(17) PRIMARY KEY,
    payload JSONB NOT NULL,
    source VARCHAR(50) NOT NULL,
    cached_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE dtc_codes (
    code VARCHAR(10) PRIMARY KEY,
    category VARCHAR(32) NOT NULL,
    title TEXT NOT NULL,
    source VARCHAR(64) NOT NULL
);

CREATE INDEX idx_sessions_vin ON diagnostic_sessions (vin, created_at DESC);
CREATE INDEX idx_resolutions_dtc ON confirmed_resolutions (diagnostic_trouble_code);
CREATE INDEX idx_resolutions_verified ON confirmed_resolutions (is_verified_fix)
    WHERE is_verified_fix;

INSERT INTO shops (id, name, location_country, location_city)
VALUES (
    '00000000-0000-4000-8000-000000000001',
    'Dev Bay',
    'Nigeria',
    'Lagos'
);

INSERT INTO technicians (id, shop_id, full_name, reputation_score)
VALUES (
    '00000000-0000-4000-8000-000000000002',
    '00000000-0000-4000-8000-000000000001',
    'Dev Technician',
    100
);

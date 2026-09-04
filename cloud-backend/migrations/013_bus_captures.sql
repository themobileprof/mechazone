-- Observed UDS/bus facts per VIN at this shop. Not a job closeout and not an invented DID map.
-- Re-scans upsert: nodes that ever answered stay ever_reachable. Other shops cannot read this row.
CREATE TABLE bus_captures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID REFERENCES shops(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    vin VARCHAR(17) NOT NULL REFERENCES vehicles(vin) ON DELETE CASCADE,
    profile TEXT NOT NULL DEFAULT '',
    adapter_type TEXT NOT NULL DEFAULT '',
    host_os TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT '',
    make_hint TEXT NOT NULL DEFAULT '',
    model_hint TEXT NOT NULL DEFAULT '',
    year_hint INT NOT NULL DEFAULT 0,
    modules JSONB NOT NULL DEFAULT '[]',
    identity JSONB NOT NULL DEFAULT '[]',
    live JSONB NOT NULL DEFAULT '[]',
    active_codes TEXT[] NOT NULL DEFAULT '{}',
    coverage JSONB NOT NULL DEFAULT '{}',
    raw_hex_excerpt TEXT NOT NULL DEFAULT '',
    scan_count INT NOT NULL DEFAULT 1,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX bus_captures_shop_vin
    ON bus_captures (shop_id, vin)
    WHERE shop_id IS NOT NULL;
CREATE UNIQUE INDEX bus_captures_freelancer_vin
    ON bus_captures (technician_id, vin)
    WHERE shop_id IS NULL;

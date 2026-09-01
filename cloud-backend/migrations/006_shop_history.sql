-- Shop-scoped vehicle work history. Jobs do not follow the VIN to other shops.
CREATE INDEX IF NOT EXISTS idx_sessions_vin_shop
    ON diagnostic_sessions (vin, shop_id, created_at DESC);

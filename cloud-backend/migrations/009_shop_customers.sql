CREATE TABLE shop_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID REFERENCES shops(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    vin VARCHAR(17) NOT NULL REFERENCES vehicles(vin) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    plate TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- This shop's file on this VIN. Freelancers (no shop) key by technician.
CREATE UNIQUE INDEX shop_customers_shop_vin
    ON shop_customers (shop_id, vin)
    WHERE shop_id IS NOT NULL;
CREATE UNIQUE INDEX shop_customers_freelancer_vin
    ON shop_customers (technician_id, vin)
    WHERE shop_id IS NULL;

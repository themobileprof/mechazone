-- Tests this shop already ran or ruled out on a VIN. Not a closeout.
-- Rebuild uses settled rows so the next playbook does not repeat a dead end.
CREATE TABLE playbook_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID REFERENCES shops(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    vin VARCHAR(17) NOT NULL REFERENCES vehicles(vin) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'test',
    title TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'done', 'ruled_out')),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX playbook_checks_shop_vin_fp
    ON playbook_checks (shop_id, vin, fingerprint)
    WHERE shop_id IS NOT NULL;
CREATE UNIQUE INDEX playbook_checks_freelancer_vin_fp
    ON playbook_checks (technician_id, vin, fingerprint)
    WHERE shop_id IS NULL;

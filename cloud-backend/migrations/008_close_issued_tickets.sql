-- Tickets whose email already has a login were left pending when ISSUE LOGIN
-- did not close the request. Safe to re-run.
UPDATE access_requests ar
SET status = 'provisioned', reviewed_at = COALESCE(ar.reviewed_at, NOW())
WHERE ar.status = 'pending'
  AND EXISTS (
    SELECT 1 FROM users u
    WHERE lower(u.email) = lower(ar.contact_email)
  );

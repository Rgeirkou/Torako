-- Ref deduplication for /stats: each upstream operation (voucher id) is
-- counted at most once, so repeated redemptions of the same voucher never
-- inflate the totals.
CREATE TABLE IF NOT EXISTS recorded_refs (
    ref         TEXT PRIMARY KEY,
    channel     TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

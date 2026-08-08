-- Failed-operation counters for /stats: every failed attempt (validation
-- or upstream failure) increments the per-channel error count.
ALTER TABLE stats_totals
    ADD COLUMN IF NOT EXISTS tw_error_count bigint NOT NULL DEFAULT 0;

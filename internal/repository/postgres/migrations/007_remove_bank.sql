-- Bank (slip verification) channel removed. Drop its statistics columns
-- and strip the stale bank scope from any existing keys.
ALTER TABLE stats_totals
    DROP COLUMN IF EXISTS bank_count,
    DROP COLUMN IF EXISTS bank_amount_cents,
    DROP COLUMN IF EXISTS bank_error_count;

UPDATE api_keys
SET scopes = array_remove(scopes, 'bank')
WHERE scopes && ARRAY['bank'];

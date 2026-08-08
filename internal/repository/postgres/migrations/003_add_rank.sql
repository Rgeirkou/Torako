-- Rate-limit tier per API key: member (default), partner, admin (unlimited).
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS rank TEXT NOT NULL DEFAULT 'member';

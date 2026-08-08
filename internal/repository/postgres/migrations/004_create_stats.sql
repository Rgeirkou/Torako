-- All-time success statistics. A single row (id=1) is updated with atomic
-- increments so concurrent instances never lose counts.
CREATE TABLE IF NOT EXISTS stats_totals (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    tw_count        BIGINT NOT NULL DEFAULT 0,
    tw_amount_cents BIGINT NOT NULL DEFAULT 0
);
INSERT INTO stats_totals (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

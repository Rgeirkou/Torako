package postgres

import (
	"context"
	"os"
	"testing"
)

func TestStatsStore_RecordAndTotals(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping postgres integration test")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE stats_totals SET tw_count = 0, tw_amount_cents = 0, tw_error_count = 0 WHERE id = 1`); err != nil {
		t.Fatalf("reset stats: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE recorded_refs`); err != nil {
		t.Fatalf("reset refs: %v", err)
	}

	store := NewStatsStore(pool)
	if err := store.RecordTw(ctx, 10000, "tw-ref-1"); err != nil {
		t.Fatalf("record tw: %v", err)
	}
	if err := store.RecordTw(ctx, 250, "tw-ref-2"); err != nil {
		t.Fatalf("record tw: %v", err)
	}
	if err := store.RecordTw(ctx, 10000, "tw-ref-1"); err != nil {
		t.Fatalf("record duplicate tw ref: %v", err)
	}

	if err := store.RecordTwError(ctx); err != nil {
		t.Fatalf("record tw error: %v", err)
	}
	if err := store.RecordTwError(ctx); err != nil {
		t.Fatalf("record tw error: %v", err)
	}

	stats, err := store.Totals(ctx)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if stats.TrueMoney.Count != 2 || stats.TrueMoney.Amount != 102.50 {
		t.Fatalf("truemoney stats = %+v, want count 2 amount 102.50", stats.TrueMoney)
	}
	if stats.Count != 2 || stats.Amount != 102.50 {
		t.Fatalf("totals = count %d amount %.2f, want 2 and 102.50", stats.Count, stats.Amount)
	}
	if stats.Errors != 2 {
		t.Fatalf("totals errors = %d, want 2", stats.Errors)
	}
	if stats.TrueMoney.Errors != 2 {
		t.Fatalf("part errors = %d, want 2", stats.TrueMoney.Errors)
	}
}

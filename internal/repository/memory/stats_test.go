package memory

import (
	"context"
	"testing"
)

func TestStatsStore_RecordAndTotals(t *testing.T) {
	store := NewStatsStore()
	ctx := context.Background()

	stats, err := store.Totals(ctx)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if stats.Count != 0 || stats.Amount != 0 || stats.Errors != 0 || stats.TrueMoney.Count != 0 {
		t.Fatalf("fresh store must be zero, got %+v", stats)
	}

	if err := store.RecordTwError(ctx); err != nil {
		t.Fatalf("record tw error: %v", err)
	}
	if err := store.RecordTwError(ctx); err != nil {
		t.Fatalf("record tw error: %v", err)
	}

	if err := store.RecordTw(ctx, 10000, "tw-ref-1"); err != nil {
		t.Fatalf("record tw: %v", err)
	}
	if err := store.RecordTw(ctx, 250, "tw-ref-2"); err != nil {
		t.Fatalf("record tw: %v", err)
	}
	if err := store.RecordTw(ctx, 100000000, "tw-ref-1"); err != nil {
		t.Fatalf("record duplicate tw ref: %v", err)
	}

	stats, err = store.Totals(ctx)
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

package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
)

func TestKeyStore(t *testing.T) {
	store := NewKeyStore()
	ctx := context.Background()

	key := &model.ApiKey{Name: "a", KeyHash: "h1", Scopes: []string{"tw"}}
	if err := store.Create(ctx, key); err != nil {
		t.Fatalf("create: %v", err)
	}
	if key.ID != 1 {
		t.Fatalf("id = %d, want 1", key.ID)
	}
	if key.Rank != "member" {
		t.Fatalf("rank = %q, want default member", key.Rank)
	}

	if err := store.Create(ctx, &model.ApiKey{KeyHash: "h1"}); err != model.ErrConflict {
		t.Fatalf("dup hash: got %v, want ErrConflict", err)
	}

	got, err := store.GetByHash(ctx, "h1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "a" || len(got.Scopes) != 1 {
		t.Fatalf("unexpected key: %+v", got)
	}

	if _, err := store.GetByHash(ctx, "missing"); err != model.ErrNotFound {
		t.Fatalf("missing: got %v, want ErrNotFound", err)
	}

	if err := store.Touch(ctx, 1); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if err := store.Revoke(ctx, 1); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := store.Revoke(ctx, 99); err != model.ErrNotFound {
		t.Fatalf("revoke missing: got %v, want ErrNotFound", err)
	}

	after, _ := store.GetByHash(ctx, "h1")
	if after.RevokedAt == nil || after.LastUsedAt == nil || after.RequestCount != 1 {
		t.Fatalf("revoke/touch not applied: %+v", after)
	}

	keys, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("list len = %d, want 1", len(keys))
	}
}

func TestKeyStore_Concurrent(t *testing.T) {
	store := NewKeyStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_ = store.Create(ctx, &model.ApiKey{KeyHash: "h-" + time.Now().String() + string(rune(i))})
		}(i)
		go func() {
			defer wg.Done()
			_ = store.Touch(ctx, 1)
			_, _ = store.List(ctx)
		}()
	}
	wg.Wait()

	keys, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 50 {
		t.Fatalf("keys = %d, want 50", len(keys))
	}
}

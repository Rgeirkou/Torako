package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
)

func testPool(t *testing.T) *KeyStore {
	t.Helper()
	return newStore(t, true)
}

func newStore(t *testing.T, truncate bool) *KeyStore {
	t.Helper()

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

	if truncate {
		if _, err := pool.Exec(ctx, `TRUNCATE api_keys RESTART IDENTITY`); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}

	return NewKeyStore(pool)
}

func TestPostgresKeyStore(t *testing.T) {
	store := testPool(t)
	ctx := context.Background()

	key := &model.ApiKey{Name: "a", KeyHash: "hash-1", Scopes: []string{"tw"}}
	if err := store.Create(ctx, key); err != nil {
		t.Fatalf("create: %v", err)
	}
	if key.ID == 0 || key.CreatedAt.IsZero() {
		t.Fatalf("create did not populate id/created_at: %+v", key)
	}

	if err := store.Create(ctx, &model.ApiKey{Name: "dup", KeyHash: "hash-1"}); err != model.ErrConflict {
		t.Fatalf("duplicate: got %v, want ErrConflict", err)
	}

	got, err := store.GetByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "a" || got.Rank != "member" || len(got.Scopes) != 1 || got.Scopes[0] != "tw" || got.RequestCount != 0 {
		t.Fatalf("unexpected key: %+v", got)
	}

	partner := &model.ApiKey{Name: "p", Rank: "partner", KeyHash: "hash-2", Scopes: []string{"tw"}}
	if err := store.Create(ctx, partner); err != nil {
		t.Fatalf("create partner: %v", err)
	}
	if partner.Rank != "partner" {
		t.Fatalf("partner rank = %q, want partner", partner.Rank)
	}

	if _, err := store.GetByHash(ctx, "nope"); err != model.ErrNotFound {
		t.Fatalf("missing: got %v, want ErrNotFound", err)
	}

	if err := store.Touch(ctx, key.ID); err != nil {
		t.Fatalf("touch: %v", err)
	}
	got, _ = store.GetByHash(ctx, "hash-1")
	if got.RequestCount != 1 || got.LastUsedAt == nil {
		t.Fatalf("touch not applied: %+v", got)
	}

	if err := store.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := store.Revoke(ctx, 9999); err != model.ErrNotFound {
		t.Fatalf("revoke missing: got %v, want ErrNotFound", err)
	}
	got, _ = store.GetByHash(ctx, "hash-1")
	if got.RevokedAt == nil {
		t.Fatalf("revoke not applied: %+v", got)
	}

	keys, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("list len = %d, want 2", len(keys))
	}
}

func TestPostgresMigration_EmptyScopes(t *testing.T) {
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

	const migration = "002_fix_empty_scopes.sql"
	if _, err := pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migration); err != nil {
		t.Fatalf("reset migration marker: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE api_keys RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	insert := func(name, hash string, scopes string) {
		t.Helper()
		if _, err := pool.Exec(ctx,
			`INSERT INTO api_keys (name, key_hash, scopes) VALUES ($1, $2, $3)`, name, hash, scopes); err != nil {
			t.Fatalf("insert %s: %v", name, err)
		}
	}
	insert("legacy-empty", "hash-empty", "{}")
	insert("legacy-admin", "hash-admin", "{admin}")

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var emptyScopes []string
	if err := pool.QueryRow(ctx, `SELECT scopes FROM api_keys WHERE key_hash = 'hash-empty'`).Scan(&emptyScopes); err != nil {
		t.Fatalf("read empty key: %v", err)
	}
	if len(emptyScopes) != 1 || emptyScopes[0] != "tw" {
		t.Fatalf("empty-scope key scopes = %v, want [tw]", emptyScopes)
	}

	var adminScopes []string
	if err := pool.QueryRow(ctx, `SELECT scopes FROM api_keys WHERE key_hash = 'hash-admin'`).Scan(&adminScopes); err != nil {
		t.Fatalf("read admin key: %v", err)
	}
	if len(adminScopes) != 1 || adminScopes[0] != "admin" {
		t.Fatalf("admin-scope key must be untouched, got %v", adminScopes)
	}

	var rank string
	if err := pool.QueryRow(ctx, `SELECT rank FROM api_keys WHERE key_hash = 'hash-empty'`).Scan(&rank); err != nil {
		t.Fatalf("read rank: %v", err)
	}
	if rank != "member" {
		t.Fatalf("legacy keys must default to member rank, got %q", rank)
	}

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate again: %v", err)
	}
}

func TestPostgresKeyStore_PersistenceAcrossStores(t *testing.T) {
	store := testPool(t)
	ctx := context.Background()

	exp := time.Now().Add(24 * time.Hour)
	if err := store.Create(ctx, &model.ApiKey{
		Name: "persisted", KeyHash: "hash-persist", Scopes: []string{"admin"}, ExpiresAt: &exp,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	second := newStore(t, false)
	got, err := second.GetByHash(ctx, "hash-persist")
	if err != nil {
		t.Fatalf("get from new store: %v", err)
	}
	if got.Name != "persisted" || got.ExpiresAt == nil {
		t.Fatalf("unexpected: %+v", got)
	}
}

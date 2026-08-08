package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/apikey"
	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/repository/memory"
)

func testKeyService() *KeyService {
	return NewKeyService(memory.NewKeyStore())
}

func TestCreateKey(t *testing.T) {
	svc := testKeyService()
	in := model.CreateApiKeyInput{Name: "client-a", Scopes: []string{"tw"}}

	view, err := svc.CreateKey(context.Background(), in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Key == "" {
		t.Fatal("view must include plaintext key")
	}
	if len(view.Key) != 43 {
		t.Fatalf("key length = %d, want 43", len(view.Key))
	}
	if view.ID == 0 || view.Name != "client-a" {
		t.Fatalf("unexpected view: %+v", view)
	}

	keys, err := svc.repo.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("store keys = %d, want 1", len(keys))
	}
	if keys[0].KeyHash == "" || keys[0].KeyHash == view.Key {
		t.Fatal("store must hold the hash, not the plaintext")
	}
}

func TestCreateKey_Validation(t *testing.T) {
	svc := testKeyService()

	cases := []model.CreateApiKeyInput{
		{Name: ""},
		{Name: "x"},
		{Name: "x", Scopes: []string{"nope"}},
		{Name: "x", Rank: "vip"},
		{Name: "x", ExpiresAt: &[]time.Time{time.Now().Add(-time.Hour)}[0]},
	}
	for i, in := range cases {
		_, err := svc.CreateKey(context.Background(), in)
		if err == nil {
			t.Fatalf("case %d: expected error", i)
		}
		var ve *model.ValidationError
		if !errors.As(err, &ve) || len(ve.Details) == 0 {
			t.Fatalf("case %d: want ValidationError with field details, got %v", i, err)
		}
	}
}

func TestListAndRevoke(t *testing.T) {
	svc := testKeyService()
	view, err := svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "a", Scopes: []string{"tw"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.RevokeKey(context.Background(), view.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if err := svc.RevokeKey(context.Background(), 999); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("revoke missing: got %v, want ErrNotFound", err)
	}

	if _, err := svc.ListKeys(context.Background()); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestCreateKey_DefaultRank(t *testing.T) {
	svc := testKeyService()
	view, err := svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "a", Scopes: []string{"tw"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Rank != "member" {
		t.Fatalf("rank = %q, want member", view.Rank)
	}

	partner, err := svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "p", Rank: "partner", Scopes: []string{"tw"}})
	if err != nil {
		t.Fatalf("create partner: %v", err)
	}
	if partner.Rank != "partner" {
		t.Fatalf("rank = %q, want partner", partner.Rank)
	}
}

func TestRotateKey(t *testing.T) {
	svc := testKeyService()
	view, err := svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "a", Rank: "partner", Scopes: []string{"tw"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rotated, err := svc.RotateKey(context.Background(), view.ID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if rotated.Key == view.Key {
		t.Fatal("rotated key must differ")
	}
	if rotated.Rank != "partner" {
		t.Fatalf("rotated rank = %q, want partner preserved", rotated.Rank)
	}

	old, err := svc.repo.GetByHash(context.Background(), hashOf(t, view.Key))
	if err != nil {
		t.Fatalf("get old: %v", err)
	}
	if old.RevokedAt == nil {
		t.Fatal("old key must be revoked")
	}

	if _, err := svc.RotateKey(context.Background(), 999); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("rotate missing: got %v, want ErrNotFound", err)
	}
}

func TestBootstrap(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("with seed key", func(t *testing.T) {
		svc := testKeyService()
		if err := svc.Bootstrap(context.Background(), "seed-key", logger); err != nil {
			t.Fatalf("bootstrap: %v", err)
		}
		keys, _ := svc.repo.List(context.Background())
		if len(keys) != 1 || keys[0].Name != "bootstrap" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
		if _, err := svc.repo.GetByHash(context.Background(), hashOf(t, "seed-key")); err != nil {
			t.Fatalf("seed key not registered: %v", err)
		}
	})

	t.Run("seed key registered even when store not empty", func(t *testing.T) {
		svc := testKeyService()
		svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "manual", Scopes: []string{"tw"}})
		if err := svc.Bootstrap(context.Background(), "seed-key", logger); err != nil {
			t.Fatalf("bootstrap: %v", err)
		}
		keys, _ := svc.repo.List(context.Background())
		if len(keys) != 2 {
			t.Fatalf("seed key must be added to non-empty store, keys=%d", len(keys))
		}
		if err := svc.Bootstrap(context.Background(), "seed-key", logger); err != nil {
			t.Fatalf("bootstrap again: %v", err)
		}
		keys, _ = svc.repo.List(context.Background())
		if len(keys) != 2 {
			t.Fatalf("bootstrap must be idempotent, keys=%d", len(keys))
		}
	})

	t.Run("no seed skips non-empty store", func(t *testing.T) {
		svc := testKeyService()
		svc.CreateKey(context.Background(), model.CreateApiKeyInput{Name: "manual", Scopes: []string{"tw"}})
		if err := svc.Bootstrap(context.Background(), "", logger); err != nil {
			t.Fatalf("bootstrap: %v", err)
		}
		keys, _ := svc.repo.List(context.Background())
		if len(keys) != 1 {
			t.Fatalf("bootstrap must skip non-empty store, keys=%d", len(keys))
		}
	})
}

func hashOf(t *testing.T, key string) string {
	t.Helper()
	return apikey.Hash(key)
}

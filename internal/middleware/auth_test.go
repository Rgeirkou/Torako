package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/apikey"
	"github.com/rgeirkou/tyrako/internal/model"
)

type fakeKeyStore struct {
	byHash map[string]*model.ApiKey
	touch  chan int64
}

func (f *fakeKeyStore) GetByHash(ctx context.Context, hash string) (*model.ApiKey, error) {
	key, ok := f.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *key
	return &cp, nil
}

func (f *fakeKeyStore) Touch(ctx context.Context, id int64) error {
	if f.touch != nil {
		f.touch <- id
	}
	return nil
}

func newAuthStore() *fakeKeyStore {
	now := time.Now()
	return &fakeKeyStore{byHash: map[string]*model.ApiKey{
		apikey.Hash("valid-key"): {ID: 1, Name: "test", KeyHash: apikey.Hash("valid-key"), Scopes: []string{"tw"}},
		apikey.Hash("expired-key"): {
			ID: 2, Name: "expired", KeyHash: apikey.Hash("expired-key"),
			Scopes: []string{"tw"}, ExpiresAt: &now,
		},
		apikey.Hash("revoked-key"): {
			ID: 3, Name: "revoked", KeyHash: apikey.Hash("revoked-key"),
			Scopes: []string{"tw"}, RevokedAt: &now,
		},
	}}
}

func TestAuth(t *testing.T) {
	store := newAuthStore()
	store.touch = make(chan int64, 10)
	handler := Auth(store, slog.New(slog.NewTextHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := APIKeyFromContext(r.Context())
		if !ok || key.Name != "test" {
			t.Fatalf("expected api key in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("missing key", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "nope")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "valid-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
		}
		select {
		case id := <-store.touch:
			if id != 1 {
				t.Fatalf("touch id = %d, want 1", id)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("expected async touch")
		}
	})

	t.Run("expired key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "expired-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("revoked key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "revoked-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusForbidden)
		}
	})
}

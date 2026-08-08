package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rgeirkou/tyrako/internal/model"
)

func TestRequireScope(t *testing.T) {
	handler := RequireScope("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("no key in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("missing scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), apiKeyCtxKey, &model.ApiKey{Scopes: []string{"tw"}})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("has scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), apiKeyCtxKey, &model.ApiKey{Scopes: []string{"admin"}})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("empty scopes denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), apiKeyCtxKey, &model.ApiKey{})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want %d: a key with no scopes must be denied, not granted every scope", rec.Code, http.StatusForbidden)
		}
	})
}

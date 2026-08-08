package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/rgeirkou/tyrako/internal/model"
)

type mockKeyService struct {
	createFn func(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error)
	listFn   func(ctx context.Context) ([]model.ApiKeyView, error)
	revokeFn func(ctx context.Context, id int64) error
	rotateFn func(ctx context.Context, id int64) (model.ApiKeyView, error)
}

func (m *mockKeyService) CreateKey(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error) {
	return m.createFn(ctx, in)
}
func (m *mockKeyService) ListKeys(ctx context.Context) ([]model.ApiKeyView, error) {
	return m.listFn(ctx)
}
func (m *mockKeyService) RevokeKey(ctx context.Context, id int64) error {
	return m.revokeFn(ctx, id)
}
func (m *mockKeyService) RotateKey(ctx context.Context, id int64) (model.ApiKeyView, error) {
	return m.rotateFn(ctx, id)
}

func newKeyHandler(m *mockKeyService) *KeyHandler {
	return NewKeyHandler(m)
}

func TestKeyHandler_Create(t *testing.T) {
	now := time.Now()
	h := newKeyHandler(&mockKeyService{
		createFn: func(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error) {
			return model.ApiKeyView{
				ID: 1, Name: in.Name, Key: "plaintext-key", Scopes: in.Scopes,
				ExpiresAt: in.ExpiresAt, CreatedAt: now,
			}, nil
		},
	})

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "valid", body: `{"name":"a","scopes":["tw"]}`, wantStatus: http.StatusCreated},
		{name: "malformed", body: `{`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/keys", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got %d, want %d, body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}

	t.Run("service error maps", func(t *testing.T) {
		h := newKeyHandler(&mockKeyService{
			createFn: func(ctx context.Context, in model.CreateApiKeyInput) (model.ApiKeyView, error) {
				return model.ApiKeyView{}, errors.New("boom")
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/keys", strings.NewReader(`{"name":"a"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Create(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got %d, want 500", rec.Code)
		}
	})
}

func TestKeyHandler_List(t *testing.T) {
	h := newKeyHandler(&mockKeyService{
		listFn: func(ctx context.Context) ([]model.ApiKeyView, error) {
			return []model.ApiKeyView{{ID: 1, Name: "a"}}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/keys", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"name":"a"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestKeyHandler_Revoke(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		h := newKeyHandler(&mockKeyService{
			revokeFn: func(ctx context.Context, id int64) error { return nil },
		})
		rec := doRequest(t, h.Revoke, http.MethodDelete, "/keys/{id}", "/keys/1")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("got %d, want 204", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := newKeyHandler(&mockKeyService{
			revokeFn: func(ctx context.Context, id int64) error { return model.ErrNotFound },
		})
		rec := doRequest(t, h.Revoke, http.MethodDelete, "/keys/{id}", "/keys/1")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("got %d, want 404", rec.Code)
		}
	})

	t.Run("bad id", func(t *testing.T) {
		h := newKeyHandler(&mockKeyService{
			revokeFn: func(ctx context.Context, id int64) error { return nil },
		})
		rec := doRequest(t, h.Revoke, http.MethodDelete, "/keys/{id}", "/keys/abc")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d, want 400", rec.Code)
		}
	})
}

func TestKeyHandler_Rotate(t *testing.T) {
	h := newKeyHandler(&mockKeyService{
		rotateFn: func(ctx context.Context, id int64) (model.ApiKeyView, error) {
			return model.ApiKeyView{ID: id, Name: "a", Key: "new-key"}, nil
		},
	})

	rec := doRequest(t, h.Rotate, http.MethodPost, "/keys/{id}/rotate", "/keys/1/rotate")

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"key":"new-key"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func doRequest(t *testing.T, fn http.HandlerFunc, method, pattern, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Method(method, pattern, fn)
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

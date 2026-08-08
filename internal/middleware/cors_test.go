package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	handler := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("preflight allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/tw", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Fatalf("ACAO = %q, want the origin", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
			t.Fatal("missing Allow-Methods on preflight")
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Fatalf("credentials must be disabled, got %q", got)
		}
	})

	t.Run("preflight disallowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/tw", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("ACAO = %q, want empty for disallowed origin", got)
		}
	})

	t.Run("actual request allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tw", nil)
		req.Header.Set("Origin", "https://app.example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Fatalf("ACAO = %q, want the origin", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Fatalf("credentials must be disabled, got %q", got)
		}
	})
}

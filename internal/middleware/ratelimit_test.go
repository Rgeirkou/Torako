package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/ratelimit"
)

func minuteWindow() time.Duration { return time.Minute }

func TestRateLimit(t *testing.T) {
	member := ratelimit.NewMemory(2, minuteWindow())
	partner := ratelimit.NewMemory(5, minuteWindow())
	handler := RateLimit(member, partner, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	authed := func(id int64, rank string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), apiKeyCtxKey, &model.ApiKey{ID: id, Rank: rank})
		return req.WithContext(ctx)
	}

	t.Run("member per key budget", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authed(1, "member"))
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200", i, rec.Code)
			}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, authed(1, "member"))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429", rec.Code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header on 429")
		}
	})

	t.Run("member default rank", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, authed(1, ""))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("empty rank must use member budget, got %d", rec.Code)
		}
	})

	t.Run("other member key unaffected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, authed(2, "member"))
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rec.Code)
		}
	})

	t.Run("partner has own higher budget", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authed(10, "partner"))
			if rec.Code != http.StatusOK {
				t.Fatalf("partner request %d: got %d, want 200", i, rec.Code)
			}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, authed(10, "partner"))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("partner over budget: got %d, want 429", rec.Code)
		}
	})

	t.Run("admin unlimited", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authed(1, "admin"))
			if rec.Code != http.StatusOK {
				t.Fatalf("admin request %d: got %d, want 200", i, rec.Code)
			}
		}
	})

	t.Run("ip fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rec.Code)
		}
	})
}

func TestRateLimitIP(t *testing.T) {
	limiter := ratelimit.NewMemory(2, minuteWindow())
	handler := RateLimitIP(limiter, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	reqFrom := func(xff string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		return req
	}

	t.Run("per ip budget", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, reqFrom("1.2.3.4"))
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200", i, rec.Code)
			}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, reqFrom("1.2.3.4"))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429", rec.Code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header on 429")
		}
	})

	t.Run("other ip unaffected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, reqFrom("5.6.7.8"))
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rec.Code)
		}
	})

	t.Run("last xff entry wins", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, reqFrom("9.9.9.9, 8.8.8.8"))
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d, want 200 (different last IP)", rec.Code)
		}
	})

	t.Run("spoofed first entry cannot bypass the ip budget", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, reqFrom("8.8.8.8, 1.2.3.4"))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429: the attacker-supplied first entry must be ignored and the real client IP (last entry) enforced", rec.Code)
		}
	})
}

func TestClientIP(t *testing.T) {
	mk := func(xff, remote string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		req.RemoteAddr = remote
		return req
	}

	t.Run("trust proxy reads last xff entry", func(t *testing.T) {
		if got := clientIP(mk("203.0.113.9, 10.0.0.1", "127.0.0.1:1234"), true); got != "10.0.0.1" {
			t.Fatalf("got %q, want the last XFF entry (appended by the trusted proxy)", got)
		}
	})

	t.Run("trust proxy single entry", func(t *testing.T) {
		if got := clientIP(mk("203.0.113.9", "127.0.0.1:1234"), true); got != "203.0.113.9" {
			t.Fatalf("got %q, want the single XFF entry", got)
		}
	})

	t.Run("no trust ignores xff", func(t *testing.T) {
		if got := clientIP(mk("203.0.113.9", "127.0.0.1:1234"), false); got != "127.0.0.1" {
			t.Fatalf("got %q, want RemoteAddr host", got)
		}
	})

	t.Run("trust with no xff falls back", func(t *testing.T) {
		if got := clientIP(mk("", "127.0.0.1:1234"), true); got != "127.0.0.1" {
			t.Fatalf("got %q, want RemoteAddr host", got)
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	for _, header := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Cache-Control",
	} {
		if rec.Header().Get(header) == "" {
			t.Fatalf("missing security header %s", header)
		}
	}
}

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rgeirkou/tyrako/internal/config"
)

const testAPIKey = "test-api-key-1234567890"

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	return newTestRouterCfg(&config.Config{AllowOrigins: []string{}, BootstrapAPIKey: testAPIKey})
}

func newTestRouterCfg(cfg *config.Config) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if cfg.BootstrapAPIKey == "" {
		cfg.BootstrapAPIKey = testAPIKey
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 35 * time.Second
	}
	router, _ := NewRouter(cfg, logger, nil)
	return router
}

func withKey(req *http.Request) *http.Request {
	req.Header.Set("X-API-Key", testAPIKey)
	return req
}

func TestRouter_IPLimitBeforeAuth(t *testing.T) {
	router := newTestRouterCfg(&config.Config{
		AllowOrigins:       []string{},
		BootstrapAPIKey:    testAPIKey,
		RateLimitEnabled:   false,
		RateLimitIPEnabled: true,
		RateLimitIPMax:     2,
		RateLimitIPWindow:  time.Minute,
	})

	keyless := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/tw/0812345678/abc123", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < 2; i++ {
		if rec := keyless(); rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: got %d, want 401 (IP limit must not preempt auth before its budget is spent), body=%s",
				i, rec.Code, rec.Body.String())
		}
	}

	rec := keyless()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429 after IP budget exhausted", rec.Code)
	}
}

func TestRouter_RequiresAPIKey(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/tw/0812345678/abc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/tw/0812345678/abc123", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d for invalid key, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouter_Stats(t *testing.T) {
	router := newTestRouter(t)

	t.Run("public without api key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/stats", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var body struct {
			Data struct {
				Amount    float64 `json:"amount"`
				Count     int64   `json:"count"`
				Errors    int64   `json:"errors"`
				TrueMoney struct {
					Amount float64 `json:"amount"`
					Count  int64   `json:"count"`
					Errors int64   `json:"errors"`
				} `json:"truemoney"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Data.Count != 0 || body.Data.Amount != 0 || body.Data.Errors != 0 {
			t.Fatalf("fresh stats must be zero, got %+v", body.Data)
		}
		if body.Data.TrueMoney.Count != 0 || body.Data.TrueMoney.Errors != 0 {
			t.Fatalf("fresh part stats must be zero, got %+v", body.Data)
		}
	})
}

func TestRouter_UnknownRoute(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestRouter_TwGet_EncodedGiftLink(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/campaign/vouchers/abc123/redeem" {
			t.Fatalf("got path %q, want /campaign/vouchers/abc123/redeem", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":{"code":"SUCCESS","message":"SUCCESS"},"data":{"voucher":{"voucher_id":"1","amount_baht":"100.00","full_name":"Test"}}}`))
	}))
	defer upstream.Close()

	router := newTestRouterCfg(&config.Config{AllowOrigins: []string{}, TrueMoneyBaseURL: upstream.URL})

	req := httptest.NewRequest(http.MethodGet,
		"/tw/0812345678/https%3A%2F%2Fgift.truemoney.com%2Fcampaign%2F%3Fv%3Dabc123", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, withKey(req))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"SUCCESS"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestRouter_TwPost_InvalidInput(t *testing.T) {
	router := newTestRouter(t)

	body := strings.NewReader(`{"phone":"123","gift":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/tw", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, withKey(req))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"details"`) {
		t.Fatalf("body should include details: %s", rec.Body.String())
	}
}

func TestRouter_RateLimit(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":{"code":"SUCCESS","message":"SUCCESS"}}`))
	}))
	defer upstream.Close()

	router := newTestRouterCfg(&config.Config{
		AllowOrigins:        []string{},
		BootstrapAPIKey:     testAPIKey,
		TrueMoneyBaseURL:    upstream.URL,
		RateLimitEnabled:    true,
		RateLimitMemberMax:  2,
		RateLimitPartnerMax: 10,
		RateLimitWindow:     time.Minute,
	})

	createKey := func(name, rank string) string {
		t.Helper()
		body := strings.NewReader(fmt.Sprintf(`{"name":%q,"rank":%q,"scopes":["tw"]}`, name, rank))
		req := httptest.NewRequest(http.MethodPost, "/keys", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAPIKey)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s: got %d, body=%s", name, rec.Code, rec.Body.String())
		}
		var created struct {
			Data struct {
				Key string `json:"key"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		return created.Data.Key
	}

	redeem := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/tw/0812345678/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", nil)
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("member limited to 2/min", func(t *testing.T) {
		key := createKey("member-a", "member")
		for i := 0; i < 2; i++ {
			rec := redeem(key)
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200, body=%s", i, rec.Code, rec.Body.String())
			}
		}
		rec := redeem(key)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429, body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header")
		}
	})

	t.Run("partner limited to 10/min", func(t *testing.T) {
		key := createKey("partner-a", "partner")
		for i := 0; i < 10; i++ {
			rec := redeem(key)
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200, body=%s", i, rec.Code, rec.Body.String())
			}
		}
		if rec := redeem(key); rec.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429", rec.Code)
		}
	})

	t.Run("admin unlimited", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			rec := redeem(testAPIKey)
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: got %d, want 200, body=%s", i, rec.Code, rec.Body.String())
			}
		}
	})
}

func TestRouter_KeyManagement(t *testing.T) {
	router := newTestRouter(t)

	t.Run("requires admin scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/keys", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("create list revoke rotate", func(t *testing.T) {
		body := strings.NewReader(`{"name":"client-a","scopes":["tw"]}`)
		req := httptest.NewRequest(http.MethodPost, "/keys", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAPIKey)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var created struct {
			Data struct {
				ID  int64  `json:"id"`
				Key string `json:"key"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("unmarshal response: %v, body=%s", err, rec.Body.String())
		}
		if created.Data.Key == "" || created.Data.ID == 0 {
			t.Fatalf("expected key and id in response, body=%s", rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/keys", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"client-a"`) {
			t.Fatalf("list: status=%d body=%s", rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodDelete, "/keys/"+strconv.FormatInt(created.Data.ID, 10), nil)
		req.Header.Set("X-API-Key", testAPIKey)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("revoke: got status %d, body=%s", rec.Code, rec.Body.String())
		}
	})
}

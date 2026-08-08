package redislimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func testLimiter(t *testing.T, max int, window time.Duration) (*Limiter, *redis.Client) {
	t.Helper()

	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set, skipping redis integration test")
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush db: %v", err)
	}

	return New(client, max, window), client
}

func TestAllow_WithinLimit(t *testing.T) {
	l, _ := testLimiter(t, 2, time.Minute)
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k1"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	l, _ := testLimiter(t, 2, time.Minute)
	for i := 0; i < 2; i++ {
		l.Allow("k1")
	}
	ok, retry := l.Allow("k1")
	if ok {
		t.Fatal("request over limit must be rejected")
	}
	if retry <= 0 || retry > time.Minute {
		t.Fatalf("retryAfter = %v, want 0 < retry <= window", retry)
	}
}

func TestAllow_PerKeyIsolation(t *testing.T) {
	l, _ := testLimiter(t, 1, time.Minute)
	l.Allow("a")
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("different keys must have independent buckets")
	}
}

func TestAllow_WindowExpiry(t *testing.T) {
	l, _ := testLimiter(t, 1, 2*time.Second)
	l.Allow("k1")
	if ok, _ := l.Allow("k1"); ok {
		t.Fatal("over limit within window")
	}
	time.Sleep(3 * time.Second)
	if ok, _ := l.Allow("k1"); !ok {
		t.Fatal("window must reset after expiry")
	}
}

func TestAllow_SubSecondWindowClamped(t *testing.T) {
	l, _ := testLimiter(t, 1, 100*time.Millisecond)
	if ok, _ := l.Allow("k1"); !ok {
		t.Fatal("first request must be allowed")
	}
	ok, _ := l.Allow("k1")
	if ok {
		t.Fatal("second request within the window must be rejected — if the key expired immediately (TTL 0) the count would have reset")
	}
}

func TestAllow_FallsBackToMemoryOnRedisDown(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
		MaxRetries:  0,
	})
	t.Cleanup(func() { _ = client.Close() })

	l := New(client, 1, time.Minute)
	if ok, _ := l.Allow("k1"); !ok {
		t.Fatal("first request must be allowed via the in-memory fallback")
	}
	if ok, _ := l.Allow("k1"); ok {
		t.Fatal("second request must be rejected by the in-memory fallback — the limit must not silently disappear when redis is unreachable")
	}
	if ok, _ := l.Allow("k2"); !ok {
		t.Fatal("different keys must keep independent fallback buckets")
	}
}

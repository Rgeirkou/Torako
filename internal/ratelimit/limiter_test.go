package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

func TestAllow_WithinLimit(t *testing.T) {
	l := NewMemory(2, time.Minute)
	for i := 0; i < 2; i++ {
		ok, _ := l.Allow("k1")
		if !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	l := NewMemory(2, time.Minute)
	if ok, _ := l.Allow("k1"); !ok {
		t.Fatal("request 1 should be allowed")
	}
	if ok, _ := l.Allow("k1"); !ok {
		t.Fatal("request 2 should be allowed (at the limit)")
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
	l := NewMemory(1, time.Minute)
	l.Allow("a")
	ok, _ := l.Allow("b")
	if !ok {
		t.Fatal("different keys must have independent buckets")
	}
}

func TestAllow_WindowReset(t *testing.T) {
	l := NewMemory(1, time.Minute)
	base := time.Now()
	l.now = func() time.Time { return base }

	l.Allow("k1")
	if ok, _ := l.Allow("k1"); ok {
		t.Fatal("over limit within window")
	}

	l.now = func() time.Time { return base.Add(61 * time.Second) }
	ok, _ := l.Allow("k1")
	if !ok {
		t.Fatal("window must reset after expiry")
	}
}

func TestAllow_PurgesStaleBuckets(t *testing.T) {
	l := NewMemory(1, time.Minute)
	base := time.Now()
	l.now = func() time.Time { return base }
	l.maxBuckets = 2

	l.Allow("a")
	l.Allow("b")
	l.now = func() time.Time { return base.Add(2 * time.Minute) }
	l.Allow("c")

	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 live bucket after purge, got %d", n)
	}
}

func TestAllow_Concurrent(t *testing.T) {
	l := NewMemory(1000, time.Minute)
	done := make(chan bool)
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				l.Allow(fmt.Sprintf("g%d", j))
			}
			done <- true
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

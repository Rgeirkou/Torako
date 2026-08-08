// Package ratelimit provides rate limiting with swappable backends.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter decides whether a request for a bucket key is allowed.
type Limiter interface {
	// Allow reports whether a request for key is within the limit. When
	// false, the returned duration is the time to wait before retrying.
	Allow(key string) (bool, time.Duration)
}

type entry struct {
	count   int
	resetAt time.Time
}

// MemoryLimiter is an in-memory fixed-window limiter, suitable for a single
// instance. Use the Redis backend for multi-instance deployments.
type MemoryLimiter struct {
	mu         sync.Mutex
	max        int
	window     time.Duration
	now        func() time.Time
	buckets    map[string]*entry
	maxBuckets int
}

func NewMemory(max int, window time.Duration) *MemoryLimiter {
	return &MemoryLimiter{
		max:        max,
		window:     window,
		now:        time.Now,
		buckets:    make(map[string]*entry),
		maxBuckets: 10000,
	}
}

func (l *MemoryLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxBuckets {
			l.purge(now)
			if len(l.buckets) >= l.maxBuckets {
				l.evictOldest()
			}
		}
		e = &entry{resetAt: now.Add(l.window)}
		l.buckets[key] = e
	}

	if now.After(e.resetAt) || now.Equal(e.resetAt) {
		e.count = 0
		e.resetAt = now.Add(l.window)
	}

	e.count++
	if e.count > l.max {
		return false, time.Until(e.resetAt)
	}
	return true, 0
}

// purge drops buckets whose window has expired.
func (l *MemoryLimiter) purge(now time.Time) {
	for key, entry := range l.buckets {
		if now.After(entry.resetAt) {
			delete(l.buckets, key)
		}
	}
}

// evictOldest drops the bucket that resets soonest, bounding memory even when
// every bucket is still within its window.
func (l *MemoryLimiter) evictOldest() {
	var oldestKey string
	var oldestAt time.Time
	for key, entry := range l.buckets {
		if oldestKey == "" || entry.resetAt.Before(oldestAt) {
			oldestKey = key
			oldestAt = entry.resetAt
		}
	}
	if oldestKey != "" {
		delete(l.buckets, oldestKey)
	}
}

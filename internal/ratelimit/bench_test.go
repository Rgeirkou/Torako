package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkMemoryLimiter_Allow_SingleKey(b *testing.B) {
	l := NewMemory(1000, time.Minute)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = l.Allow("key:1")
	}
}

func BenchmarkMemoryLimiter_Allow_ManyKeys(b *testing.B) {
	l := NewMemory(1000, time.Minute)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = l.Allow("key:" + fmt.Sprint(i%10000))
	}
}

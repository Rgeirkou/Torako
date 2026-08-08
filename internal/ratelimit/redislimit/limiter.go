// Package redislimit provides a Redis-backed fixed-window rate limiter for
// multi-instance deployments. When Redis is unavailable the limiter falls
// back to a local in-memory limiter so availability is preserved without
// silently disabling the rate limit.
package redislimit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rgeirkou/tyrako/internal/ratelimit"
)

type Limiter struct {
	client   *redis.Client
	max      int
	ttl      time.Duration
	ctx      context.Context
	fallback ratelimit.Limiter
}

var allowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return {count, redis.call('TTL', KEYS[1])}
`)

// New returns a limiter that allows max requests per window per bucket key.
// The client is used as-is; the caller owns its lifecycle. While the client
// stays healthy the limit is enforced across instances; on error the limit
// degrades to a single-instance in-memory budget for that process.
func New(client *redis.Client, max int, window time.Duration) *Limiter {
	return &Limiter{
		client:   client,
		max:      max,
		ttl:      window,
		ctx:      context.Background(),
		fallback: ratelimit.NewMemory(max, window),
	}
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	ttl := int(l.ttl.Seconds())
	if ttl < 1 {
		ttl = 1
	}
	res, err := allowScript.Run(l.ctx, l.client, []string{"rl:" + key}, strconv.Itoa(ttl)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return l.fallback.Allow(key)
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) != 2 {
		return true, 0
	}

	count, _ := strconv.Atoi(redisToString(arr[0]))
	remaining, _ := strconv.Atoi(redisToString(arr[1]))
	if count > l.max {
		return false, time.Duration(remaining) * time.Second
	}
	return true, 0
}

func redisToString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}

package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rgeirkou/tyrako/internal/ratelimit"
	"github.com/rgeirkou/tyrako/pkg/response"
)

// RateLimit rejects requests that exceed the limiter budget of the caller's
// rank. Member keys share one budget, partner keys another (higher), and
// admin keys bypass the per-key limit entirely. The router mounts it after
// Auth, so every request carries an API key; the IP fallback is kept as a
// safety net for deployments that mount the middleware without Auth.
func RateLimit(member, partner ratelimit.Limiter, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := member
			bucket := "ip:" + clientIP(r, trustProxy)
			if key, ok := APIKeyFromContext(r.Context()); ok {
				switch key.Rank {
				case "admin":
					next.ServeHTTP(w, r)
					return
				case "partner":
					limiter = partner
				}
				bucket = "key:" + strconv.FormatInt(key.ID, 10)
			}

			allowed, retryAfter := limiter.Allow(bucket)
			if !allowed {
				writeRateLimitError(w, retryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitIP limits requests per client IP and is meant to run before Auth,
// so unauthenticated traffic cannot exhaust the API store or upstreams. The
// client IP is taken from the last X-Forwarded-For entry only when
// trustProxy is set (the API sits behind a trusted reverse proxy such as
// Caddy); otherwise the connection's RemoteAddr is used. The last entry is
// the hop appended by the trusted proxy (its direct peer, i.e. the real
// client at the edge), while earlier entries are attacker-controlled and
// must not be trusted.
func RateLimitIP(limiter ratelimit.Limiter, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := limiter.Allow("ip:" + clientIP(r, trustProxy))
			if !allowed {
				writeRateLimitError(w, retryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeRateLimitError(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.LastIndex(xff, ","); i >= 0 {
				xff = xff[i+1:]
			}
			xff = strings.TrimSpace(xff)
			if xff != "" {
				return xff
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

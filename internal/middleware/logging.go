package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// sensitivePathPrefixes are GET routes that embed credentials in the path
// (e.g. /tw/{phone}/{gift} where the gift code is money-equivalent).
// Their remaining segments are masked in logs.
var sensitivePathPrefixes = []string{"/tw/"}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", redactPath(r.URL.Path),
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", chimw.GetReqID(r.Context()),
			)
		})
	}
}

func redactPath(p string) string {
	for _, prefix := range sensitivePathPrefixes {
		if strings.HasPrefix(p, prefix) && len(p) > len(prefix) {
			return prefix + "***"
		}
	}
	return p
}

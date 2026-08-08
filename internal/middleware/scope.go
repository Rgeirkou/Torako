package middleware

import (
	"net/http"

	"github.com/rgeirkou/tyrako/pkg/response"
)

func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, ok := APIKeyFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "missing api key")
				return
			}
			if !hasScope(key.Scopes, scope) {
				response.Error(w, http.StatusForbidden, "insufficient scope")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasScope reports whether scopes contains required. Keys with no scopes are
// denied: an empty scope list grants nothing (fail closed), so a key created
// or edited outside the normal API flow can never escalate to every scope.
func hasScope(scopes []string, required string) bool {
	if len(scopes) == 0 {
		return false
	}
	for _, s := range scopes {
		if s == required {
			return true
		}
	}
	return false
}

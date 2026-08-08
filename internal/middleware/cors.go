package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORS allows the configured origins. Credentials are disabled because
// authentication uses the X-API-Key header (no cookies), which also keeps
// ALLOW_ORIGINS=* valid for public browser access.
func CORS(allowOrigins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}

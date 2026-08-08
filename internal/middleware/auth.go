package middleware

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/rgeirkou/tyrako/internal/apikey"
	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/pkg/response"
)

type apiKeyStore interface {
	GetByHash(ctx context.Context, hash string) (*model.ApiKey, error)
	Touch(ctx context.Context, id int64) error
}

type ctxKey string

const apiKeyCtxKey ctxKey = "api_key"

func Auth(store apiKeyStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				w.Header().Set("WWW-Authenticate", "ApiKey")
				response.Error(w, http.StatusUnauthorized, "missing api key")
				return
			}

			hash := apikey.Hash(key)
			rec, err := store.GetByHash(r.Context(), hash)
			if err != nil || subtle.ConstantTimeCompare([]byte(rec.KeyHash), []byte(hash)) != 1 {
				response.Error(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			if rec.RevokedAt != nil || (rec.ExpiresAt != nil && !rec.ExpiresAt.After(time.Now())) {
				response.Error(w, http.StatusForbidden, "api key revoked or expired")
				return
			}

			ctx := context.WithValue(r.Context(), apiKeyCtxKey, rec)
			go touchAsync(store, logger, rec.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func touchAsync(store apiKeyStore, logger *slog.Logger, id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Touch(ctx, id); err != nil {
		logger.Warn("touch api key usage", "key_id", id, "err", err)
	}
}

func APIKeyFromContext(ctx context.Context) (*model.ApiKey, bool) {
	key, ok := ctx.Value(apiKeyCtxKey).(*model.ApiKey)
	return key, ok
}

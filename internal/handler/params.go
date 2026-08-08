package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func pathParam(r *http.Request, key string) (string, error) {
	raw := chi.URLParam(r, key)
	dec, err := url.PathUnescape(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s parameter", key)
	}
	return dec, nil
}

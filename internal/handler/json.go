package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rgeirkou/tyrako/pkg/response"
)

const maxBodySize = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return err
	}
	return nil
}

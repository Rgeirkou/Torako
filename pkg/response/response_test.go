package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusCreated, map[string]string{"id": "1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusCreated)
	}
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if env.Data == nil || env.Error != "" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()

	Error(rec, http.StatusUnauthorized, "unauthorized")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if env.Data != nil || env.Error != "unauthorized" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

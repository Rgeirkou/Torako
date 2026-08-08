package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{in: "/tw/0812345678/ABCDEFGHIJKLMNOPQRSTUVWXYZ", want: "/tw/***"},
		{in: "/tw", want: "/tw"},
		{in: "/keys", want: "/keys"},
		{in: "/", want: "/"},
	}
	for _, c := range cases {
		if got := redactPath(c.in); got != c.want {
			t.Fatalf("redactPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLogging_RedactsSensitivePath(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tw/0812345678/ABCDEFGHIJKLMNOPQRSTUVWXYZ", nil))

	log := buf.String()
	if !strings.Contains(log, "path=/tw/***") {
		t.Fatalf("log must contain the redacted path, got: %s", log)
	}
	if strings.Contains(log, "0812345678") || strings.Contains(log, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("log must not leak the phone or gift code, got: %s", log)
	}
}

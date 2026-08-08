package apikey

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	k1, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	k2, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if k1 == k2 {
		t.Fatal("two generated keys must differ")
	}
	if len(k1) != 43 {
		t.Fatalf("key length = %d, want 43", len(k1))
	}
}

func TestHash_Deterministic(t *testing.T) {
	h1 := Hash("secret-key")
	h2 := Hash("secret-key")
	h3 := Hash("other-key")

	if h1 != h2 {
		t.Fatal("hash must be deterministic")
	}
	if h1 == h3 {
		t.Fatal("different keys must produce different hashes")
	}
	if !strings.HasPrefix(h1, "2a80f1fa0455b1a33a29a4a3edcbd5b9df38ac6811a70cbb80f15357ff5b8b2e") && len(h1) != 64 {
		t.Fatalf("hash should be 64 hex chars, got %q", h1)
	}
}

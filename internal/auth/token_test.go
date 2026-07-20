package auth

import (
	"crypto/subtle"
	"testing"
)

func TestGenerateProducesDistinctPrefixedTokens(t *testing.T) {
	first, err := Generate("mjs_", 32)
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}
	second, err := Generate("mjs_", 32)
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}
	if first == second {
		t.Fatal("generated tokens must be unique")
	}
	if len(first) <= len("mjs_") || first[:len("mjs_")] != "mjs_" {
		t.Fatalf("token %q does not have the expected prefix", first)
	}
}

func TestHashIsStableLowercaseSHA256(t *testing.T) {
	const plain = "mjs_example-token"
	first := Hash(plain)
	second := Hash(plain)
	if first != second {
		t.Fatal("hash must be stable")
	}
	if len(first) != 64 {
		t.Fatalf("expected 64 hexadecimal characters, got %d", len(first))
	}
	if subtle.ConstantTimeCompare([]byte(first), []byte(second)) != 1 {
		t.Fatal("equal digests must compare successfully")
	}
}

func TestPrefixDoesNotRevealWholeToken(t *testing.T) {
	const plain = "mjs_0123456789abcdefghijklmnopqrstuvwxyz"
	prefix := Prefix(plain)
	if prefix == "" || prefix == plain {
		t.Fatalf("unsafe token prefix %q", prefix)
	}
	if len(prefix) > 12 {
		t.Fatalf("prefix is too long: %d", len(prefix))
	}
}

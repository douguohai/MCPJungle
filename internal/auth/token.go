package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const visibleTokenPrefixLength = 12

// Generate returns a high-entropy URL-safe token. Only the returned plaintext
// should be shown to the caller; persistent storage must use Hash instead.
func Generate(prefix string, randomBytes int) (string, error) {
	if randomBytes < 16 {
		return "", fmt.Errorf("token entropy must be at least 16 bytes")
	}
	b := make([]byte, randomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the lowercase hexadecimal SHA-256 digest used for lookups.
func Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Prefix returns a short display-only fragment that helps users identify a
// token without exposing enough material to authenticate.
func Prefix(plain string) string {
	if len(plain) <= visibleTokenPrefixLength {
		return plain
	}
	return plain[:visibleTokenPrefixLength]
}

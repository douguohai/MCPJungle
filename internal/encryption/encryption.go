// Package encryption provides AES-256-GCM encryption for sensitive fields
// (e.g. bearer tokens) using a master key supplied via MCPHUB_MASTER_KEY.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	masterKey []byte
	mu        sync.RWMutex

	// ErrMasterKeyNotSet is returned when an encryption/decryption operation
	// is attempted without a configured master key.
	ErrMasterKeyNotSet = errors.New("MCPHUB_MASTER_KEY is not set")
)

// SetMasterKey parses the hex-encoded master key and stores it.
// The key must decode to exactly 32 bytes (AES-256).
func SetMasterKey(hexKey string) error {
	if hexKey == "" {
		return fmt.Errorf("master key must not be empty")
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return fmt.Errorf("master key must be valid hex: %w", err)
	}
	if len(key) != 32 {
		return fmt.Errorf("master key must be 32 bytes (got %d)", len(key))
	}
	mu.Lock()
	masterKey = key
	mu.Unlock()
	return nil
}

// IsSet reports whether the master key has been configured.
func IsSet() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(masterKey) == 32
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// The returned string is hex-encoded: nonce || ciphertext.
func Encrypt(plaintext string) (string, error) {
	mu.RLock()
	key := masterKey
	mu.RUnlock()
	if len(key) == 0 {
		return "", ErrMasterKeyNotSet
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	// Store nonce + ciphertext together so Decrypt can recover them.
	combined := append(nonce, ciphertext...)
	return hex.EncodeToString(combined), nil
}

// Decrypt decrypts a hex-encoded nonce+ciphertext string produced by Encrypt.
func Decrypt(encoded string) (string, error) {
	mu.RLock()
	key := masterKey
	mu.RUnlock()
	if len(key) == 0 {
		return "", ErrMasterKeyNotSet
	}

	data, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext (not hex): %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	return string(plaintext), nil
}

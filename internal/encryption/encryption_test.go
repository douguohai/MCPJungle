package encryption

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// 32 bytes in hex = 64 hex chars
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	plaintext := "my-secret-bearer-token"
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext must differ from plaintext")
	}

	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptEmptyString(t *testing.T) {
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	ciphertext, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty string, got %q", decrypted)
	}
}

func TestDecryptInvalidHex(t *testing.T) {
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	_, err := Decrypt("not-hex")
	if err == nil {
		t.Fatal("expected error for non-hex input")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	ciphertext, err := Encrypt("hello")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	SetMasterKey("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	_, err = Decrypt(ciphertext)
	if err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestSetMasterKeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"empty", "", true},
		{"not hex", "zzzz", true},
		{"too short hex", "abcd", true},
		{"valid 32-byte hex", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetMasterKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetMasterKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestDifferentCiphertextsSamePlaintext(t *testing.T) {
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	c1, _ := Encrypt("same")
	c2, _ := Encrypt("same")
	if c1 == c2 {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestLongPlaintext(t *testing.T) {
	SetMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	plaintext := strings.Repeat("x", 10000)
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt long: %v", err)
	}
	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt long: %v", err)
	}
	if decrypted != plaintext {
		t.Fatal("round-trip failed for long plaintext")
	}
}

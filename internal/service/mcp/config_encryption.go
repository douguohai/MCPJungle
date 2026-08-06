package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/encryption"
)

// EncryptConfigBearerToken encrypts the bearer_token field in a raw config JSON
// blob. If the field is absent or empty the JSON is returned unchanged.
// Only AES-256-GCM is used when the master key is configured; otherwise this is
// a no-op (dev mode without MCPHUB_MASTER_KEY).
func EncryptConfigBearerToken(config []byte) ([]byte, error) {
	if !encryption.IsSet() {
		return config, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return nil, fmt.Errorf("unmarshal config for encryption: %w", err)
	}
	token, _ := m["bearer_token"].(string)
	if token == "" {
		return config, nil
	}
	enc, err := encryption.Encrypt(token)
	if err != nil {
		return nil, fmt.Errorf("encrypt bearer_token: %w", err)
	}
	m["bearer_token"] = enc
	return json.Marshal(m)
}

// DecryptConfigBearerToken decrypts the bearer_token field in a raw config JSON
// blob. If the master key is not set the config is returned unchanged.
func DecryptConfigBearerToken(config []byte) ([]byte, error) {
	if !encryption.IsSet() {
		return config, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return nil, fmt.Errorf("unmarshal config for decryption: %w", err)
	}
	token, _ := m["bearer_token"].(string)
	if token == "" {
		return config, nil
	}
	dec, err := encryption.Decrypt(token)
	if err != nil {
		return nil, fmt.Errorf("decrypt bearer_token: %w", err)
	}
	m["bearer_token"] = dec
	return json.Marshal(m)
}

// MaskConfigBearerToken replaces the bearer_token value in a raw config JSON
// blob with a masked summary ("***abcd") and adds a has_token boolean.
// The has_token field indicates whether a token was originally present.
func MaskConfigBearerToken(config []byte) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return nil, fmt.Errorf("unmarshal config for masking: %w", err)
	}
	token, _ := m["bearer_token"].(string)
	if token == "" {
		m["has_token"] = false
		return json.Marshal(m)
	}
	m["has_token"] = true
	if len(token) > 4 {
		m["bearer_token"] = "***" + token[len(token)-4:]
	} else {
		m["bearer_token"] = "***"
	}
	return json.Marshal(m)
}

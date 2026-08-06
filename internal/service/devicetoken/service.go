// Package devicetoken manages DeviceToken records used to authenticate MCP
// proxy requests.  It replaces the legacy McpClient model.
//
// The raw token has the form `mcpdt_<dbID>_<secret>`.  Only the prefix
// (mcpdt_<dbID>) and the SHA-256 of the secret are persisted, so a leaked
// database cannot be used to impersonate a device.  The raw secret is returned
// to the caller exactly once at creation time.
package devicetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// DefaultTokenTTL matches design doc §8 (90-day default device token lifetime).
const DefaultTokenTTL = 90 * 24 * time.Hour

const secretBytes = 32 // → 64 hex chars

// Service manages DeviceToken records.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Create issues a new device token. restrictedServers is only used when
// scopeMode is DeviceTokenScopeRestricted.  Returns the raw token string
// (to be shown to the user once) and the persisted record.
func (s *Service) Create(userID uint, name, scopeMode string, restrictedServers []uint, ttl time.Duration) (string, *model.DeviceToken, error) {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	if scopeMode == "" {
		scopeMode = model.DeviceTokenScopeInheritAll
	}
	exp := time.Now().UTC().Add(ttl)
	tok := &model.DeviceToken{
		UserID:    userID,
		Name:      name,
		TokenPrefix: "pending",
		TokenHash:   "pending",
		ScopeMode:   scopeMode,
		Status:      model.DeviceTokenStatusActive,
		ExpiresAt:   &exp,
	}
	if err := s.db.Create(tok).Error; err != nil {
		return "", nil, err
	}

	// Generate secret + final prefix using the DB id.
	secret, err := generateSecret()
	if err != nil {
		return "", nil, err
	}
	prefix := fmt.Sprintf("mcpdt_%d", tok.ID)
	raw := prefix + "_" + secret
	tok.TokenPrefix = prefix
	tok.TokenHash = hashSecret(secret)
	if err := s.db.Save(tok).Error; err != nil {
		return "", nil, err
	}

	// Record restricted service grants.
	if scopeMode == model.DeviceTokenScopeRestricted {
		for _, sid := range restrictedServers {
			if err := s.db.Create(&model.DeviceTokenService{
				DeviceTokenID: tok.ID,
				McpServerID:   sid,
			}).Error; err != nil {
				return "", nil, err
			}
		}
	}
	return raw, tok, nil
}

// GetByID retrieves a token by its database primary key.
func (s *Service) GetByID(id uint) (*model.DeviceToken, error) {
	var tok model.DeviceToken
	if err := s.db.First(&tok, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("token not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &tok, nil
}

// GetByToken validates a raw token string and returns the active DeviceToken.
// Returns gorm.ErrRecordNotFound on any failure (bad format, not found,
// revoked, expired, or secret mismatch) — the error is deliberately opaque so
// callers cannot distinguish which step failed.
func (s *Service) GetByToken(raw string) (*model.DeviceToken, error) {
	prefix, secret, ok := parseRawToken(raw)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var tok model.DeviceToken
	if err := s.db.Where("token_prefix = ?", prefix).First(&tok).Error; err != nil {
		return nil, gorm.ErrRecordNotFound
	}
	if tok.Status != model.DeviceTokenStatusActive {
		return nil, gorm.ErrRecordNotFound
	}
	if tok.ExpiresAt != nil && time.Now().UTC().After(*tok.ExpiresAt) {
		return nil, gorm.ErrRecordNotFound
	}
	if subtle.ConstantTimeCompare([]byte(hashSecret(secret)), []byte(tok.TokenHash)) != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return &tok, nil
}

// GetRestrictedServices returns the MCP server IDs a restricted-scope token is
// allowed to use.  For inherit_all tokens the result is irrelevant (the caller
// should use permission.UserEffectiveServices instead).
func (s *Service) GetRestrictedServices(tokenID uint) ([]uint, error) {
	var ids []uint
	err := s.db.Model(&model.DeviceTokenService{}).
		Where("device_token_id = ?", tokenID).
		Pluck("mcp_server_id", &ids).Error
	return ids, err
}

// Revoke sets a token's status to revoked (idempotent).
func (s *Service) Revoke(tokenID uint) error {
	return s.db.Model(&model.DeviceToken{}).Where("id = ?", tokenID).
		Updates(map[string]interface{}{
			"status":     model.DeviceTokenStatusRevoked,
			"revoked_at": time.Now().UTC(),
		}).Error
}

// List returns device tokens visible to the caller: all tokens for admins,
// own tokens only for regular users.
func (s *Service) List(userID uint, role types.UserRole) ([]model.DeviceToken, error) {
	var tokens []model.DeviceToken
	q := s.db
	if role != types.UserRoleSystemAdmin {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// Delete removes a token.  Non-admins can only delete their own.
func (s *Service) Delete(tokenID, userID uint, role types.UserRole) error {
	q := s.db.Unscoped().Where("id = ?", tokenID)
	if role != types.UserRoleSystemAdmin {
		q = q.Where("user_id = ?", userID)
	}
	result := q.Delete(&model.DeviceToken{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("token not found: %w", apierrors.ErrNotFound)
	}
	return result.Error
}

// --- helpers ----------------------------------------------------------------

func generateSecret() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// parseRawToken splits `mcpdt_<id>_<secret>` into (prefix, secret, ok).
// The secret is the portion after the last underscore (hex, no underscores).
func parseRawToken(raw string) (prefix, secret string, ok bool) {
	if !strings.HasPrefix(raw, "mcpdt_") {
		return "", "", false
	}
	idx := strings.LastIndex(raw, "_")
	if idx <= len("mcpdt_") {
		return "", "", false
	}
	return raw[:idx], raw[idx+1:], true
}

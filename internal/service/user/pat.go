package user

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
)

// patPrefixLen is how many chars of the plaintext token are stored as a
// human-readable Prefix (e.g. to help users identify a token in a list).
const patPrefixLen = 8

// hashToken returns the lowercase hex sha256 of a plaintext token. PATs are
// looked up by hash so the plaintext is never stored or queried directly.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreatePAT issues a new personal access token for the user. The plaintext
// token is returned once; only its sha256 hash is persisted.
func (u *UserService) CreatePAT(userID uint, name string, expiresAt *time.Time) (string, *model.PersonalAccessToken, error) {
	plaintext, err := internal.GenerateAccessToken()
	if err != nil {
		return "", nil, err
	}
	prefix := plaintext
	if len(prefix) > patPrefixLen {
		prefix = prefix[:patPrefixLen]
	}
	pat := &model.PersonalAccessToken{
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		TokenHash: hashToken(plaintext),
		ExpiresAt: expiresAt,
	}
	if err := u.db.Create(pat).Error; err != nil {
		return "", nil, fmt.Errorf("failed to create personal access token: %w", err)
	}
	return plaintext, pat, nil
}

// ListPATs returns the user's tokens ordered by creation time (hash excluded
// by json:"-" on the model fields that matter; here we return the rows as-is).
func (u *UserService) ListPATs(userID uint) ([]model.PersonalAccessToken, error) {
	var pats []model.PersonalAccessToken
	if err := u.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&pats).Error; err != nil {
		return nil, fmt.Errorf("failed to list personal access tokens: %w", err)
	}
	return pats, nil
}

// RevokePAT soft-revokes a token by id, scoped to the owning user.
func (u *UserService) RevokePAT(userID, id uint) error {
	res := u.db.Model(&model.PersonalAccessToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return fmt.Errorf("failed to revoke token: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("token not found: %w", apierrors.ErrNotFound)
	}
	return nil
}

// GetUserByPAT resolves a plaintext PAT to its user. Returns ErrNotFound if the
// token is unknown, revoked, or expired. Updates LastUsedAt on success.
func (u *UserService) GetUserByPAT(token string) (*model.User, error) {
	var pat model.PersonalAccessToken
	err := u.db.Where("token_hash = ?", hashToken(token)).First(&pat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("token not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	if pat.RevokedAt != nil {
		return nil, fmt.Errorf("token revoked: %w", apierrors.ErrNotFound)
	}
	if pat.ExpiresAt != nil && pat.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired: %w", apierrors.ErrNotFound)
	}
	// best-effort last-used update
	_ = u.db.Model(&pat).Update("last_used_at", time.Now()).Error

	var user model.User
	if err := u.db.First(&user, pat.UserID).Error; err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	return &user, nil
}

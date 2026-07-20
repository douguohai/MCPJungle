// Package auth issues and verifies JWT-based session tokens for human users.
//
// Human users authenticate with a username+password and receive a short-lived
// JWT (no refresh token: once it expires they must log in again). MCP client
// (machine) authentication is separate and unchanged.
package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// JwtSecretEnvVar is the environment variable holding the HMAC secret used to
// sign session JWTs. Required in enterprise mode; optional in dev mode (an
// ephemeral random secret is generated when unset).
const JwtSecretEnvVar = "MCPJUNGLE_JWT_SECRET"

// TokenTTL is how long a session JWT remains valid. There is no refresh token,
// so users must re-authenticate after this elapses.
const TokenTTL = 8 * time.Hour

// Claims are the custom JWT claims embedded in each session token.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Signer signs and parses session JWTs using an HMAC-SHA256 secret.
type Signer struct {
	secret []byte
}

// NewSigner returns a Signer that uses the given secret.
func NewSigner(secret []byte) *Signer {
	return &Signer{secret: secret}
}

// ResolveSecret reads the JWT signing secret from the environment.
// In dev mode an unset secret yields a random ephemeral one (valid only for
// the process lifetime). In enterprise mode an unset secret is a fatal error.
func ResolveSecret(devMode bool) ([]byte, error) {
	if s := os.Getenv(JwtSecretEnvVar); s != "" {
		return []byte(s), nil
	}
	if devMode {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed to generate ephemeral JWT secret: %w", err)
		}
		return b, nil
	}
	return nil, errors.New(JwtSecretEnvVar + " must be set when running in enterprise mode")
}

// Sign issues a session JWT for the user and returns the token and its expiry.
func (s *Signer) Sign(u *model.User) (string, time.Time, error) {
	if u == nil {
		return "", time.Time{}, errors.New("cannot sign token for nil user")
	}
	exp := time.Now().Add(TokenTTL)
	claims := Claims{
		Username: u.Username,
		Role:     string(u.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", u.ID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign session token: %w", err)
	}
	return signed, exp, nil
}

// Parse validates the token string and returns its claims. It rejects tokens
// signed with any method other than HMAC.
func (s *Signer) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

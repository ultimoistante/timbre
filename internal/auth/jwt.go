package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ultimoistante/timbre/internal/models"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the JWT payload carried by both token types.
type Claims struct {
	UserID uint        `json:"uid"`
	Role   models.Role `json:"role"`
	Type   TokenType   `json:"typ"`
	jwt.RegisteredClaims
}

// Manager issues and verifies JWTs using a shared HMAC secret.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager builds a token Manager.
func NewManager(secret []byte, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// Issue creates a signed token of the given type for the user.
func (m *Manager) Issue(u *models.User, typ TokenType) (string, error) {
	ttl := m.accessTTL
	if typ == RefreshToken {
		ttl = m.refreshTTL
	}

	now := time.Now()
	claims := Claims{
		UserID: u.ID,
		Role:   u.Role,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse verifies a token string and returns its claims.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// AccessTTL exposes the access-token lifetime (used for cookie max-age).
func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

// RefreshTTL exposes the refresh-token lifetime.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

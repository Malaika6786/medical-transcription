package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

const currentSchemaVersion = 2

// JWTManager handles JWT token creation and validation.
type JWTManager struct {
	baseSecretKey   []byte
	secretKey       []byte
	tokenDuration   time.Duration
	tokenVersion    int64
	versionIssuedAt time.Time
	mu              sync.RWMutex
}

// Claims represents JWT claims. SchemaVersion lets ValidateToken reject tokens
// issued before a schema migration (schema < 2 → old single-role tokens).
type Claims struct {
	UserID        string       `json:"userId"`
	Email         string       `json:"email"`
	Roles         []string     `json:"roles"`
	Permissions   []Permission `json:"permissions"`
	SchemaVersion int          `json:"schema_version"`
	jwt.RegisteredClaims
}

// HasPermission returns true if p is present in the claims' effective permission list.
func (c *Claims) HasPermission(p Permission) bool {
	for _, ep := range c.Permissions {
		if ep == p {
			return true
		}
	}
	return false
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		baseSecretKey:   []byte(secretKey),
		secretKey:       []byte(secretKey),
		tokenDuration:   tokenDuration,
		tokenVersion:    1,
		versionIssuedAt: time.Now(),
	}
}

// GenerateToken creates a new JWT for a user, embedding effective permissions computed
// from the provided role map. Permissions are pre-computed here so every downstream
// check is stateless (no store lookup per request).
func (m *JWTManager) GenerateToken(user *User, rolesMap map[string]*Role) (string, error) {
	m.mu.RLock()
	secretKey := m.secretKey
	m.mu.RUnlock()

	claims := &Claims{
		UserID:        user.ID,
		Email:         user.Email,
		Roles:         user.Roles,
		Permissions:   user.EffectivePermissions(rolesMap),
		SchemaVersion: currentSchemaVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xstek-medical-transcription",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateToken validates a JWT token and returns the claims.
// Tokens with SchemaVersion < 2 are rejected to force re-login after migration.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	m.mu.RLock()
	secretKey := m.secretKey
	versionIssuedAt := m.versionIssuedAt
	m.mu.RUnlock()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Reject tokens issued before the last force-logout.
	if claims.IssuedAt != nil && claims.IssuedAt.Time.Before(versionIssuedAt) {
		return nil, ErrExpiredToken
	}

	// Reject pre-migration tokens (old single-role schema).
	if claims.SchemaVersion < currentSchemaVersion {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// InvalidateAllTokens rotates the secret key to invalidate all existing tokens.
func (m *JWTManager) InvalidateAllTokens() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return err
	}
	suffix := hex.EncodeToString(randomBytes)
	m.secretKey = append(m.baseSecretKey, []byte(suffix)...)
	m.tokenVersion++
	m.versionIssuedAt = time.Now()
	return nil
}

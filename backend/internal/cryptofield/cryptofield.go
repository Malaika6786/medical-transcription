// Package cryptofield provides application-level field encryption for the
// most sensitive columns in the schema (patient NHS number, name, date of
// birth — see internal/pgstore/schema.sql). This exists because Postgres
// itself stores those columns as plain TEXT: disk/volume encryption is a
// hosting concern outside this repo's control, and per the SystmOne
// integration feasibility report (SYSTMONE_INTEGRATION_REPORT.md), "no
// encryption at rest" was flagged as a concrete DTAC/DSPT blocker.
//
// Ciphertext format: base64( nonce(12 bytes) || AES-256-GCM(nonce, plaintext) ).
// There is no key-rotation scheme here — this is a single static key from
// FIELD_ENCRYPTION_KEY, which is the right amount of complexity for the
// project's current scale. If key rotation is ever needed, prepend a key-id
// byte to the ciphertext before shipping this to production with real
// patient data.
package cryptofield

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrKeyNotConfigured is returned by NewFromEnv-style constructors when no
// encryption key is available. Callers (pgstore) should treat this as fatal
// for any code path that touches patients — see main.go's boot-time check.
var ErrKeyNotConfigured = errors.New("cryptofield: FIELD_ENCRYPTION_KEY is not configured")

// Cipher encrypts/decrypts individual field values and computes a
// deterministic lookup hash for values that need to be matched (e.g. NHS
// number uniqueness) without ever decrypting every row to compare.
type Cipher struct {
	gcm     cipher.AEAD
	hmacKey []byte
}

// New builds a Cipher from a 32-byte key (AES-256). Use NewFromHexKey to
// parse the FIELD_ENCRYPTION_KEY env var, which is stored as hex.
func New(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("cryptofield: key must be 32 bytes (got %d) — generate one with `openssl rand -hex 32`", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cryptofield: init AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cryptofield: init GCM: %w", err)
	}
	// The HMAC key is derived from the same secret via SHA-256 rather than
	// requiring a second env var — this is a lookup-hash key, not a second
	// encryption key, so domain-separating it from the raw AES key this way
	// is sufficient.
	sum := sha256.Sum256(append([]byte("cryptofield-hmac-key:"), key...))
	return &Cipher{gcm: gcm, hmacKey: sum[:]}, nil
}

// NewFromHexKey parses a hex-encoded 32-byte key, as produced by
// `openssl rand -hex 32` (the value FIELD_ENCRYPTION_KEY should hold).
func NewFromHexKey(hexKey string) (*Cipher, error) {
	if hexKey == "" {
		return nil, ErrKeyNotConfigured
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("cryptofield: FIELD_ENCRYPTION_KEY must be hex-encoded: %w", err)
	}
	return New(key)
}

// Encrypt returns ciphertext for plaintext, or ("", nil) if plaintext is
// empty (so an optional field like date-of-birth can stay genuinely absent
// rather than becoming an encrypted empty string).
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("cryptofield: generate nonce: %w", err)
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Returns ("", nil) for empty ciphertext.
func (c *Cipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("cryptofield: ciphertext is not valid base64: %w", err)
	}
	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("cryptofield: ciphertext too short")
	}
	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("cryptofield: decrypt failed (wrong key, or data tampered): %w", err)
	}
	return string(plaintext), nil
}

// LookupHash returns a deterministic HMAC-SHA256 (hex) of value, suitable
// for a unique index or an exact-match WHERE clause without ever decrypting
// stored ciphertext to compare it. Not reversible.
func (c *Cipher) LookupHash(value string) string {
	if value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, c.hmacKey)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

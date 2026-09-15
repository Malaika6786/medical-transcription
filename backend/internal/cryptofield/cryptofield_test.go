package cryptofield

import (
	"strings"
	"testing"
)

func testCipher(t *testing.T) *Cipher {
	t.Helper()
	// 32 zero bytes hex-encoded (64 hex chars) — a fixed test key, never
	// used for real data.
	c, err := NewFromHexKey(strings.Repeat("00", 32))
	if err != nil {
		t.Fatalf("NewFromHexKey: %v", err)
	}
	return c
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c := testCipher(t)
	plaintext := "9434765919"
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext must not equal plaintext")
	}
	decrypted, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt(Encrypt(%q)) = %q", plaintext, decrypted)
	}
}

func TestEncryptDecrypt_Empty(t *testing.T) {
	c := testCipher(t)
	ciphertext, err := c.Encrypt("")
	if err != nil || ciphertext != "" {
		t.Errorf("Encrypt(\"\") = (%q, %v), want (\"\", nil)", ciphertext, err)
	}
	plaintext, err := c.Decrypt("")
	if err != nil || plaintext != "" {
		t.Errorf("Decrypt(\"\") = (%q, %v), want (\"\", nil)", plaintext, err)
	}
}

func TestLookupHash_Deterministic(t *testing.T) {
	c := testCipher(t)
	h1 := c.LookupHash("9434765919")
	h2 := c.LookupHash("9434765919")
	if h1 != h2 {
		t.Errorf("LookupHash is not deterministic: %q != %q", h1, h2)
	}
	if h3 := c.LookupHash("9434765920"); h3 == h1 {
		t.Error("LookupHash produced the same hash for two different values")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	c1, _ := New(make([]byte, 32))
	otherKey := make([]byte, 32)
	otherKey[0] = 1
	c2, _ := New(otherKey)

	ciphertext, err := c1.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := c2.Decrypt(ciphertext); err == nil {
		t.Error("Decrypt with the wrong key should fail, but it succeeded")
	}
}

// Command mktoken mints a short-lived JWT for local API testing.
// Usage: JWT_SECRET=... go run ./cmd/mktoken [-user <userID>] [-email <email>]
// Pass -user to impersonate a real user (e.g. to exercise /api/search, whose
// results are scoped to the token's user ID).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"corti-backend/internal/auth"
)

func main() {
	userID := flag.String("user", "e2e-test", "user ID to embed in the token")
	email := flag.String("email", "admin@xstek.net", "email to embed in the token")
	flag.Parse()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "JWT_SECRET is required")
		os.Exit(1)
	}

	claims := &auth.Claims{
		UserID:        *userID,
		Email:         *email,
		Roles:         []string{"superuser"},
		Permissions:   []auth.Permission{auth.PermAmbientAccess},
		SchemaVersion: 2,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xstek-medical-transcription",
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(token)
}

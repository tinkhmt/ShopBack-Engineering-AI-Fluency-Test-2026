package util

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testTokenManager(ttl time.Duration) *TokenManager {
	return NewTokenManager(strings.Repeat("k", 32), ttl)
}

func TestTokenRoundTrip(t *testing.T) {
	manager := testTokenManager(time.Minute)

	raw, err := manager.Issue(42)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	userID, err := manager.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
}

func TestTokenRejectsExpired(t *testing.T) {
	manager := testTokenManager(-time.Minute)

	raw, err := manager.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := manager.Parse(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenRejectsWrongSecret(t *testing.T) {
	issuer := NewTokenManager(strings.Repeat("a", 32), time.Minute)
	verifier := NewTokenManager(strings.Repeat("b", 32), time.Minute)

	raw, err := issuer.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := verifier.Parse(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenRejectsNoneAlgorithm(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "1",
		Issuer:    tokenIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building unsigned token: %v", err)
	}

	if _, err := testTokenManager(time.Minute).Parse(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("alg=none token error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenRejectsTamperedSignature(t *testing.T) {
	manager := testTokenManager(time.Minute)

	raw, err := manager.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := manager.Parse(raw + "x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("tampered token error = %v, want ErrInvalidToken", err)
	}
}

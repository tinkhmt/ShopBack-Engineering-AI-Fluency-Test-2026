package util

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLen = 8
	// MaxPasswordLen: bcrypt ignores every byte after the 72nd, so anything
	// longer must be rejected instead of silently truncated.
	MaxPasswordLen = 72

	// bcryptCost 12 is ~250ms on a modern CPU: slow enough to make offline
	// guessing expensive, fast enough for interactive login.
	bcryptCost = 12
)

var ErrPasswordTooLong = errors.New("password exceeds 72 bytes")

func HashPassword(password string) (string, error) {
	if len(password) > MaxPasswordLen {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// DummyPasswordHash is a real cost-12 hash of a throwaway string. Login
// compares against it when the email is unknown so both paths cost the same
// and response timing does not reveal whether an account exists.
const DummyPasswordHash = "$2a$12$ozCUJE8hMEe.4D2eNXNf.O8pViX525GURhc7g/gCUozIF/9VmtM2G"

package util

import (
	"errors"
	"strings"
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "correct horse battery"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password {
		t.Fatal("stored value is the plaintext password")
	}
	if !CheckPassword(hash, password) {
		t.Error("CheckPassword() = false for the correct password")
	}
	if CheckPassword(hash, "wrong password") {
		t.Error("CheckPassword() = true for a wrong password")
	}
}

func TestPasswordHashAcceptsExactly72Bytes(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 72)); err != nil {
		t.Fatalf("72-byte password rejected: %v", err)
	}
}

func TestPasswordHashRejectsOver72Bytes(t *testing.T) {
	_, err := HashPassword(strings.Repeat("a", 73))
	if !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("error = %v, want ErrPasswordTooLong", err)
	}
}

func TestDummyPasswordHashIsUsable(t *testing.T) {
	// A malformed constant would make the unknown-email path return faster than
	// a real bcrypt comparison, reintroducing the timing oracle.
	if CheckPassword(DummyPasswordHash, "anything") {
		t.Fatal("DummyPasswordHash matched an arbitrary password")
	}
}

package auth

import (
	"testing"
	"time"

	"github.com/ultimoistante/timbre/internal/models"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "s3cret" {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword(hash, "s3cret") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func newTestManager() *Manager {
	return NewManager([]byte("0123456789abcdef0123456789abcdef"), time.Minute, time.Hour)
}

func TestJWTRoundTrip(t *testing.T) {
	m := newTestManager()
	u := &models.User{ID: 42, Role: models.RoleAdmin}

	tok, err := m.Issue(u, AccessToken)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := m.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != 42 || claims.Role != models.RoleAdmin || claims.Type != AccessToken {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestJWTTampered(t *testing.T) {
	m := newTestManager()
	other := NewManager([]byte("ffffffffffffffffffffffffffffffff"), time.Minute, time.Hour)
	u := &models.User{ID: 1, Role: models.RoleUser}

	tok, _ := other.Issue(u, AccessToken)
	if _, err := m.Parse(tok); err == nil {
		t.Fatal("token signed with different secret must fail")
	}
}

func TestJWTExpired(t *testing.T) {
	m := NewManager([]byte("0123456789abcdef0123456789abcdef"), -time.Minute, time.Hour)
	u := &models.User{ID: 1, Role: models.RoleUser}
	tok, _ := m.Issue(u, AccessToken)
	if _, err := m.Parse(tok); err == nil {
		t.Fatal("expired token must fail")
	}
}

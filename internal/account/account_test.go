package account

import (
	"context"
	"path/filepath"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestInviteRegisterLogin(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	inv, err := s.CreateInvite(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !s.InviteValid(ctx, inv) {
		t.Fatal("fresh invite should be valid")
	}

	u, sess, err := s.Register(ctx, inv, "alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "alice" || sess == "" {
		t.Fatalf("bad register: %+v %q", u, sess)
	}
	if s.InviteValid(ctx, inv) {
		t.Error("invite should be consumed after use")
	}
	if _, _, err := s.Register(ctx, inv, "bob", "password123"); err != ErrBadInvite {
		t.Errorf("reused invite: want ErrBadInvite, got %v", err)
	}

	inv2, _ := s.CreateInvite(ctx, "")
	if _, _, err := s.Register(ctx, inv2, "alice", "password123"); err != ErrTakenName {
		t.Errorf("dup name: want ErrTakenName, got %v", err)
	}

	if u2 := s.UserBySession(ctx, sess); u2 == nil || u2.Username != "alice" {
		t.Error("session lookup failed")
	}

	_, ls, err := s.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if s.UserBySession(ctx, ls) == nil {
		t.Error("login session invalid")
	}
	if _, _, err := s.Login(ctx, "alice", "nope"); err != ErrBadCreds {
		t.Errorf("wrong pw: want ErrBadCreds, got %v", err)
	}

	inv3, _ := s.CreateInvite(ctx, "")
	if _, _, err := s.Register(ctx, inv3, "Bad Name", "password123"); err != ErrBadName {
		t.Errorf("bad name: want ErrBadName, got %v", err)
	}
	if _, _, err := s.Register(ctx, inv3, "okname", "short"); err != ErrWeakPass {
		t.Errorf("weak pass: want ErrWeakPass, got %v", err)
	}

	s.DeleteSession(ctx, ls)
	if s.UserBySession(ctx, ls) != nil {
		t.Error("session should be gone after logout")
	}
}

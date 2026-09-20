// Package account is the multi-user layer: invite-only registration, username +
// password login (PBKDF2, stdlib — no external deps), and server-side sessions.
// Backed by the shared SQLite file.
package account

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrBadInvite   = errors.New("invite link is invalid or already used")
	ErrTakenName   = errors.New("that username is taken")
	ErrBadName     = errors.New("username must be 3–32 chars: lowercase letters, digits, _ or -")
	ErrBadCreds    = errors.New("wrong username or password")
	ErrWeakPass    = errors.New("password must be at least 8 characters")
	usernameRegexp = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)
)

const pbkdf2Iter = 120_000

type User struct {
	ID          int64
	Username    string
	DisplayName string
	CreatedAt   time.Time
}

type Invite struct {
	Token     string
	Note      string
	CreatedAt time.Time
	Used      bool
}

type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS users (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  username     TEXT UNIQUE NOT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  pw_hash      TEXT NOT NULL,
  pw_salt      TEXT NOT NULL,
  pw_iter      INTEGER NOT NULL,
  created_at   TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS invites (
  token      TEXT PRIMARY KEY,
  note       TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL,
  used_by    INTEGER,
  used_at    TIMESTAMP
);
CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  user_id    INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL
);`

func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ---- invites ----

func (s *Store) CreateInvite(ctx context.Context, note string) (string, error) {
	tok := randToken(12)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO invites (token, note, created_at) VALUES (?, ?, ?)`,
		tok, strings.TrimSpace(note), time.Now().UTC())
	return tok, err
}

// inviteUsable reports whether an invite exists and is unused.
func (s *Store) inviteUsable(ctx context.Context, token string) (bool, error) {
	var usedBy sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT used_by FROM invites WHERE token = ?`, token).Scan(&usedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !usedBy.Valid, nil
}

// InviteValid is a read-only check for the join page.
func (s *Store) InviteValid(ctx context.Context, token string) bool {
	ok, err := s.inviteUsable(ctx, token)
	return err == nil && ok
}

func (s *Store) ListInvites(ctx context.Context) ([]Invite, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT token, note, created_at, used_by FROM invites ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invite
	for rows.Next() {
		var iv Invite
		var usedBy sql.NullInt64
		if err := rows.Scan(&iv.Token, &iv.Note, &iv.CreatedAt, &usedBy); err != nil {
			return nil, err
		}
		iv.Used = usedBy.Valid
		out = append(out, iv)
	}
	return out, rows.Err()
}

// ---- registration & login ----

// Register consumes an invite and creates a user, returning a new session token.
func (s *Store) Register(ctx context.Context, invite, username, password string) (*User, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernameRegexp.MatchString(username) {
		return nil, "", ErrBadName
	}
	if len(password) < 8 {
		return nil, "", ErrWeakPass
	}
	ok, err := s.inviteUsable(ctx, invite)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrBadInvite
	}

	hash, salt := hashPassword(password)
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (username, display_name, pw_hash, pw_salt, pw_iter, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		username, username, hash, salt, pbkdf2Iter, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, "", ErrTakenName
		}
		return nil, "", err
	}
	uid, _ := res.LastInsertId()

	// consume the invite only if still unused (guards a race)
	upd, err := tx.ExecContext(ctx,
		`UPDATE invites SET used_by = ?, used_at = ? WHERE token = ? AND used_by IS NULL`, uid, now, invite)
	if err != nil {
		return nil, "", err
	}
	if n, _ := upd.RowsAffected(); n == 0 {
		return nil, "", ErrBadInvite
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	u := &User{ID: uid, Username: username, DisplayName: username, CreatedAt: now}
	sess, err := s.newSession(ctx, uid)
	return u, sess, err
}

func (s *Store) Login(ctx context.Context, username, password string) (*User, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	var (
		u         User
		hash, slt string
		iter      int
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, pw_hash, pw_salt, pw_iter, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &hash, &slt, &iter, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrBadCreds
	}
	if err != nil {
		return nil, "", err
	}
	if !verifyPassword(password, hash, slt, iter) {
		return nil, "", ErrBadCreds
	}
	sess, err := s.newSession(ctx, u.ID)
	return &u, sess, err
}

func (s *Store) SetPassword(ctx context.Context, userID int64, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPass
	}
	hash, salt := hashPassword(newPassword)
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET pw_hash = ?, pw_salt = ?, pw_iter = ? WHERE id = ?`, hash, salt, pbkdf2Iter, userID)
	return err
}

func (s *Store) SetDisplayName(ctx context.Context, userID int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("display name cannot be empty")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET display_name = ? WHERE id = ?`, name, userID)
	return err
}

// ---- sessions ----

func (s *Store) newSession(ctx context.Context, userID int64) (string, error) {
	tok := randToken(24)
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tok, userID, now, now.Add(30*24*time.Hour))
	return tok, err
}

// UserBySession returns the user for a valid, unexpired session token, or nil.
func (s *Store) UserBySession(ctx context.Context, token string) *User {
	if token == "" {
		return nil
	}
	var u User
	var expires time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.display_name, u.created_at, s.expires_at
		   FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token = ?`, token).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.CreatedAt, &expires)
	if err != nil || time.Now().After(expires) {
		return nil
	}
	return &u
}

func (s *Store) DeleteSession(ctx context.Context, token string) {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
}

// ---- helpers ----

func hashPassword(pw string) (hashB64, saltB64 string) {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	dk, _ := pbkdf2.Key(sha256.New, pw, salt, pbkdf2Iter, 32)
	return base64.StdEncoding.EncodeToString(dk), base64.StdEncoding.EncodeToString(salt)
}

func verifyPassword(pw, hashB64, saltB64 string, iter int) bool {
	salt, err1 := base64.StdEncoding.DecodeString(saltB64)
	want, err2 := base64.StdEncoding.DecodeString(hashB64)
	if err1 != nil || err2 != nil {
		return false
	}
	dk, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(dk, want) == 1
}

func randToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

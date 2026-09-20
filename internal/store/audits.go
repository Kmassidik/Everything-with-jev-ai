// Package store persists finished audits so a result is a durable, shareable
// artifact (a permalink + a CSV export) rather than a throwaway screenful.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"jevai/internal/judge"

	_ "modernc.org/sqlite"
)

type Audits struct{ db *sql.DB }

const auditSchema = `
CREATE TABLE IF NOT EXISTS audits (
    id         TEXT PRIMARY KEY,
    pack       TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    report     TEXT NOT NULL
);`

// Open opens (and migrates) the audits store at path (may share the ledger's file).
func Open(path string) (*Audits, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), auditSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Audits{db: db}, nil
}

func (s *Audits) Close() error { return s.db.Close() }

// Save stores a report and returns its short shareable id.
func (s *Audits) Save(ctx context.Context, r *judge.Report) (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	id := newID()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO audits (id, pack, created_at, report) VALUES (?, ?, ?, ?)`,
		id, r.Pack, time.Now().UTC(), string(data)); err != nil {
		return "", err
	}
	return id, nil
}

// Get returns a saved report and when it was created.
func (s *Audits) Get(ctx context.Context, id string) (*judge.Report, time.Time, error) {
	var raw string
	var at time.Time
	if err := s.db.QueryRowContext(ctx,
		`SELECT report, created_at FROM audits WHERE id = ?`, id).Scan(&raw, &at); err != nil {
		return nil, time.Time{}, err
	}
	var r judge.Report
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, time.Time{}, err
	}
	return &r, at, nil
}

func newID() string {
	b := make([]byte, 9) // 12 url-safe chars
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

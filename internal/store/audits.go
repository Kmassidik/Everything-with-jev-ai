// Package store persists finished audits so a result is a durable, shareable
// artifact (a permalink + a CSV export) and shows up in the owner's history.
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

const auditTable = `
CREATE TABLE IF NOT EXISTS audits (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL DEFAULT 0,
    pack       TEXT NOT NULL,
    items      INTEGER NOT NULL DEFAULT 0,
    flagged    INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    report     TEXT NOT NULL
);`

// AuditMeta is a history-row summary (no full report).
type AuditMeta struct {
	ID        string
	Pack      string
	Items     int
	Flagged   int
	CreatedAt time.Time
}

func Open(path string) (*Audits, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, auditTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	// add columns to older tables that predate them (errors ignored if already present),
	// BEFORE creating any index that references them.
	for _, col := range []string{
		"ALTER TABLE audits ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE audits ADD COLUMN items INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE audits ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0",
	} {
		_, _ = db.ExecContext(ctx, col)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_audits_user ON audits(user_id, created_at)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("index: %w", err)
	}
	return &Audits{db: db}, nil
}

func (s *Audits) Close() error { return s.db.Close() }

// Save stores a report owned by userID and returns its short shareable id.
func (s *Audits) Save(ctx context.Context, userID int64, r *judge.Report) (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	flagged := 0
	for _, res := range r.Results {
		for _, f := range res.Findings {
			if f.Flag {
				flagged++
				break
			}
		}
	}
	id := newID()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO audits (id, user_id, pack, items, flagged, created_at, report) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userID, r.Pack, len(r.Results), flagged, time.Now().UTC(), string(data)); err != nil {
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

// ListByUser returns a user's recent audits (newest first).
func (s *Audits) ListByUser(ctx context.Context, userID int64, limit int) ([]AuditMeta, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, pack, items, flagged, created_at FROM audits WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
		userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditMeta
	for rows.Next() {
		var m AuditMeta
		if err := rows.Scan(&m.ID, &m.Pack, &m.Items, &m.Flagged, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func newID() string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

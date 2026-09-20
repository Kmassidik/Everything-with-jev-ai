package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver (no CGO)
)

// SQLite is a durable Ledger backed by a single-file database. Usage is stored as
// an append-only event log and rolled up on read.
type SQLite struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS usage_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user       TEXT    NOT NULL,
    tokens_in  INTEGER NOT NULL,
    tokens_out INTEGER NOT NULL,
    judgments  INTEGER NOT NULL,
    at         TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_usage_user ON usage_events(user);`

// OpenSQLite opens (and migrates) the database at path.
func OpenSQLite(path string) (*SQLite, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite is a single writer; one connection keeps writes serialized and safe.
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &SQLite{db: db}, nil
}

func (l *SQLite) Record(ctx context.Context, user string, in, out, judgments int) error {
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO usage_events (user, tokens_in, tokens_out, judgments, at) VALUES (?, ?, ?, ?, ?)`,
		user, in, out, judgments, time.Now().UTC())
	return err
}

func (l *SQLite) Usage(ctx context.Context, user string) (Usage, error) {
	var u Usage
	err := l.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens_in),0), COALESCE(SUM(tokens_out),0), COALESCE(SUM(judgments),0)
		   FROM usage_events WHERE user = ?`, user).
		Scan(&u.InputTokens, &u.OutputTokens, &u.Judgments)
	if err != nil {
		return Usage{}, err
	}
	return u, nil
}

func (l *SQLite) Close() error { return l.db.Close() }

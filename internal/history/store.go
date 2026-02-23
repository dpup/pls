package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	OutcomeAccepted = "accepted"
	OutcomeRejected = "rejected"
	OutcomeCopied   = "copied"
)

type Entry struct {
	ID        int64
	RepoID    int64
	CwdRel    string
	Intent    string
	Command   string
	Outcome   string
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrating: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS repos (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			root_path  TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS history (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id    INTEGER REFERENCES repos(id),
			cwd_rel    TEXT NOT NULL,
			intent     TEXT NOT NULL,
			command    TEXT NOT NULL,
			outcome    TEXT NOT NULL CHECK (outcome IN ('accepted', 'rejected', 'copied')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_history_repo_cwd ON history(repo_id, cwd_rel);
		CREATE INDEX IF NOT EXISTS idx_history_created ON history(created_at);
	`)
	return err
}

func (s *Store) EnsureRepo(rootPath string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO repos (root_path) VALUES (?) ON CONFLICT (root_path) DO NOTHING",
		rootPath,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		row := s.db.QueryRow("SELECT id FROM repos WHERE root_path = ?", rootPath)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) Record(repoID int64, cwdRel, intent, command, outcome string) error {
	_, err := s.db.Exec(
		"INSERT INTO history (repo_id, cwd_rel, intent, command, outcome) VALUES (?, ?, ?, ?, ?)",
		repoID, cwdRel, intent, command, outcome,
	)
	return err
}

func (s *Store) ProjectHistory(repoID int64, cwdRel string, limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, repo_id, cwd_rel, intent, command, outcome, created_at FROM history WHERE repo_id = ? AND cwd_rel = ? ORDER BY created_at DESC, id DESC LIMIT ?",
		repoID, cwdRel, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	return scanEntries(rows)
}

func (s *Store) RecentGlobal(limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, repo_id, cwd_rel, intent, command, outcome, created_at FROM history WHERE outcome = ? ORDER BY created_at DESC, id DESC LIMIT ?",
		OutcomeAccepted, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.RepoID, &e.CwdRel, &e.Intent, &e.Command, &e.Outcome, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

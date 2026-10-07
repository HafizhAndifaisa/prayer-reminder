// Package store persists user settings and cached prayer schedules in SQLite.
// It uses the pure-Go modernc.org/sqlite driver, so the application builds
// without CGO.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"prayer-reminder/internal/prayer"
)

// Store wraps the SQLite database connection.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the database at path and applies migrations.
func Open(path string) (*Store, error) {
	// _pragma settings enable foreign keys and a busy timeout for concurrent access.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates the schema if it does not already exist.
func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schedule_cache (
	date       TEXT PRIMARY KEY,
	payload    TEXT NOT NULL,
	fetched_at DATETIME NOT NULL
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Cache setting keys stored via the generic key/value helpers.
const (
	settingMethods = "methods"
)

// --- Settings ---

// GetSetting returns the stored value for key, or ok=false when unset.
func (s *Store) GetSetting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, true, nil
}

// SetSetting upserts a key/value pair.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	const q = `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	if _, err := s.db.ExecContext(ctx, q, key, value); err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

// GetJSON decodes the stored JSON value for key into dest. It returns false
// when the key is not present.
func (s *Store) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	raw, ok, err := s.GetSetting(ctx, key)
	if err != nil || !ok {
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return false, fmt.Errorf("decode setting %q: %w", key, err)
	}
	return true, nil
}

// SetJSON encodes value as JSON and stores it under key.
func (s *Store) SetJSON(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode setting %q: %w", key, err)
	}
	return s.SetSetting(ctx, key, string(raw))
}

// --- Schedule cache ---

// SaveSchedule caches a schedule keyed by its date.
func (s *Store) SaveSchedule(ctx context.Context, sched *prayer.Schedule) error {
	payload, err := json.Marshal(sched)
	if err != nil {
		return fmt.Errorf("encode schedule: %w", err)
	}
	const q = `INSERT INTO schedule_cache (date, payload, fetched_at) VALUES (?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET payload = excluded.payload, fetched_at = excluded.fetched_at`
	if _, err := s.db.ExecContext(ctx, q, sched.Date, string(payload), time.Now()); err != nil {
		return fmt.Errorf("save schedule: %w", err)
	}
	return nil
}

// LoadSchedule returns the cached schedule for a date, or ok=false when absent.
func (s *Store) LoadSchedule(ctx context.Context, date string) (*prayer.Schedule, bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM schedule_cache WHERE date = ?`, date).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("load schedule: %w", err)
	}
	var sched prayer.Schedule
	if err := json.Unmarshal([]byte(payload), &sched); err != nil {
		return nil, false, fmt.Errorf("decode schedule: %w", err)
	}
	return &sched, true, nil
}

// ClearScheduleCache removes every cached schedule. It is called when the
// location or calculation method changes so stale times are not served.
func (s *Store) ClearScheduleCache(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM schedule_cache`); err != nil {
		return fmt.Errorf("clear schedule cache: %w", err)
	}
	return nil
}

// --- Calculation methods cache ---

// SaveMethods caches the calculation method list.
func (s *Store) SaveMethods(ctx context.Context, methods []prayer.Method) error {
	return s.SetJSON(ctx, settingMethods, methods)
}

// LoadMethods returns the cached calculation method list.
func (s *Store) LoadMethods(ctx context.Context) ([]prayer.Method, bool, error) {
	var methods []prayer.Method
	ok, err := s.GetJSON(ctx, settingMethods, &methods)
	if err != nil || !ok {
		return nil, false, err
	}
	return methods, true, nil
}

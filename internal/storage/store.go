package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db              *sql.DB
	key             []byte
	unlocked        bool
	failedAttempts  int
	lockoutUntil    time.Time
	lastActivity    time.Time
	autoLockTimeout time.Duration
}

type Note struct {
	ID        string
	Path      string
	Title     string
	EyesOnly  bool
	UpdatedAt time.Time
}

type Folder struct {
	Path      string
	Collapsed bool
	CreatedAt time.Time
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Unlocked() bool {
	return s.unlocked
}

func (s *Store) SetAutoLockTimeout(timeout time.Duration) {
	s.autoLockTimeout = timeout
	s.UpdateActivity()
}

func (s *Store) UpdateActivity() {
	if s.unlocked {
		s.lastActivity = time.Now()
	}
}

func (s *Store) CheckAutoLock() bool {
	if !s.AutoLockExpired() {
		return false
	}
	s.Lock()
	return true
}

func (s *Store) AutoLockExpired() bool {
	if !s.unlocked || s.autoLockTimeout <= 0 {
		return false
	}
	return time.Since(s.lastActivity) > s.autoLockTimeout
}

func (s *Store) Lock() {
	s.unlocked = false
	s.key = nil
	s.lastActivity = time.Time{}
}

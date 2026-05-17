package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db       *sql.DB
	key      []byte
	unlocked bool
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

package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
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

func (s *Store) Migrate() error {
	stmts := []string{
		`create table if not exists vault (
			id integer primary key check (id = 1),
			password_hash text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists notes (
			id text primary key,
			path text not null unique,
			title text not null,
			nonce blob not null,
			ciphertext blob not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists folders (
			path text primary key,
			created_at datetime not null default current_timestamp
		)`,
		`create index if not exists idx_notes_updated on notes(updated_at desc)`,
		`create index if not exists idx_notes_path on notes(path)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) HasVault() (bool, error) {
	var count int
	if err := s.db.QueryRow(`select count(*) from vault where id = 1`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateVault(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`insert into vault (id, password_hash) values (1, ?)`, string(hash)); err != nil {
		return err
	}
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlock(password string) error {
	var hash string
	if err := s.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errors.New("bad vault password")
	}
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlocked() bool {
	return s.unlocked
}

func (s *Store) SaveNote(id, path, title, content string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	if err := s.ensureFolderParents(path); err != nil {
		return err
	}
	nonce, ciphertext, err := s.encrypt([]byte(content))
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`insert into notes (id, path, title, nonce, ciphertext, updated_at)
		 values (?, ?, ?, ?, ?, current_timestamp)
		 on conflict(path) do update set
		   title = excluded.title,
		   nonce = excluded.nonce,
		   ciphertext = excluded.ciphertext,
		   updated_at = current_timestamp`,
		id, path, title, nonce, ciphertext,
	)
	return err
}

func (s *Store) DeleteNote(path string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	_, err := s.db.Exec(`delete from notes where path = ?`, path)
	return err
}

func (s *Store) RenameNote(oldPath, newPath string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	oldPath = cleanStorePath(oldPath)
	newPath = cleanStorePath(newPath)
	if oldPath == "" || newPath == "" {
		return errors.New("note path is required")
	}
	if oldPath == newPath {
		return nil
	}
	if err := s.ensureFolderParents(newPath); err != nil {
		return err
	}
	result, err := s.db.Exec(`update notes set path = ?, updated_at = current_timestamp where path = ?`, newPath, oldPath)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return errors.New("vault note not found")
	}
	return nil
}

func (s *Store) SaveFolder(path string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	path = cleanStorePath(path)
	if path == "" {
		return errors.New("folder path is required")
	}
	if err := s.ensureFolderParents(path); err != nil {
		return err
	}
	_, err := s.db.Exec(`insert or ignore into folders (path) values (?)`, path)
	return err
}

func (s *Store) DeleteFolder(path string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	path = cleanStorePath(path)
	like := path + "/%"
	var children int
	if err := s.db.QueryRow(
		`select
		   (select count(*) from notes where path like ?)
		   +
		   (select count(*) from folders where path like ?)`,
		like, like,
	).Scan(&children); err != nil {
		return err
	}
	if children > 0 {
		return errors.New("folder is not empty")
	}
	_, err := s.db.Exec(`delete from folders where path = ?`, path)
	return err
}

func (s *Store) RenameFolder(oldPath, newPath string) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	oldPath = cleanStorePath(oldPath)
	newPath = cleanStorePath(newPath)
	if oldPath == "" || newPath == "" {
		return errors.New("folder path is required")
	}
	if oldPath == newPath {
		return nil
	}
	if strings.HasPrefix(newPath+"/", oldPath+"/") {
		return errors.New("cannot move a folder inside itself")
	}
	if err := s.ensureFolderParents(newPath); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`select count(*) from folders where path = ?`, oldPath).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("vault folder not found")
	}
	if err := updateVaultFolderPaths(tx, `select path from folders where path = ? or path like ?`, oldPath, newPath); err != nil {
		return err
	}
	if err := updateVaultNotePaths(tx, oldPath, newPath); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) LoadNote(path string) (Note, string, bool, error) {
	if !s.unlocked {
		return Note{}, "", false, errors.New("vault is locked")
	}
	row := s.db.QueryRow(`select id, path, title, nonce, ciphertext, updated_at from notes where path = ?`, path)
	var note Note
	var nonce, ciphertext []byte
	if err := row.Scan(&note.ID, &note.Path, &note.Title, &nonce, &ciphertext, &note.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Note{}, "", false, nil
		}
		return Note{}, "", false, err
	}
	plain, err := s.decrypt(nonce, ciphertext)
	if err != nil {
		return Note{}, "", false, err
	}
	return note, string(plain), true, nil
}

func (s *Store) ListNotes() ([]Note, error) {
	rows, err := s.db.Query(`select id, path, title, updated_at from notes order by path collate nocase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Path, &note.Title, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (s *Store) ListFolders() ([]Folder, error) {
	rows, err := s.db.Query(`select path, created_at from folders order by path collate nocase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []Folder
	for rows.Next() {
		var folder Folder
		if err := rows.Scan(&folder.Path, &folder.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (s *Store) ensureFolderParents(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == string(filepath.Separator) {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(dir))
	if clean == "." || clean == "" {
		return nil
	}
	prefix := ""
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." {
			continue
		}
		if prefix == "" {
			prefix = part
		} else {
			prefix += "/" + part
		}
		if _, err := s.db.Exec(`insert or ignore into folders (path) values (?)`, prefix); err != nil {
			return err
		}
	}
	return nil
}

func updateVaultFolderPaths(tx *sql.Tx, query, oldPath, newPath string) error {
	rows, err := tx.Query(query, oldPath, oldPath+"/%")
	if err != nil {
		return err
	}
	defer rows.Close()
	paths := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		replacement := renamedChildPath(path, oldPath, newPath)
		if _, err := tx.Exec(`update folders set path = ? where path = ?`, replacement, path); err != nil {
			return err
		}
	}
	return nil
}

func updateVaultNotePaths(tx *sql.Tx, oldPath, newPath string) error {
	rows, err := tx.Query(`select path from notes where path like ?`, oldPath+"/%")
	if err != nil {
		return err
	}
	defer rows.Close()
	paths := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, path := range paths {
		replacement := renamedChildPath(path, oldPath, newPath)
		if _, err := tx.Exec(`update notes set path = ?, updated_at = current_timestamp where path = ?`, replacement, path); err != nil {
			return err
		}
	}
	return nil
}

func renamedChildPath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	return newPath + strings.TrimPrefix(path, oldPath)
}

func cleanStorePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimPrefix(path, "/")
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}

func (s *Store) unlockWith(password string) {
	sum := sha3.Sum256([]byte(password))
	s.key = sum[:]
	s.unlocked = true
}

func (s *Store) encrypt(plain []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plain, nil), nil
}

func (s *Store) decrypt(nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("bad nonce size %d", len(nonce))
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

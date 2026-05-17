package storage

import (
	"database/sql"
	"errors"
	"strings"
)

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

func (s *Store) SetNoteEyesOnly(path string, eyesOnly bool) error {
	if !s.unlocked {
		return errors.New("vault is locked")
	}
	value := 0
	if eyesOnly {
		value = 1
	}
	result, err := s.db.Exec(`update notes set eyes_only = ?, updated_at = current_timestamp where path = ?`, value, cleanStorePath(path))
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

func (s *Store) LoadNote(path string) (Note, string, bool, error) {
	if !s.unlocked {
		return Note{}, "", false, errors.New("vault is locked")
	}
	row := s.db.QueryRow(`select id, path, title, eyes_only, nonce, ciphertext, updated_at from notes where path = ?`, path)
	var note Note
	var nonce, ciphertext []byte
	var eyesOnly any
	if err := row.Scan(&note.ID, &note.Path, &note.Title, &eyesOnly, &nonce, &ciphertext, &note.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Note{}, "", false, nil
		}
		return Note{}, "", false, err
	}
	note.EyesOnly = scanBool(eyesOnly)
	plain, err := s.decrypt(nonce, ciphertext)
	if err != nil {
		return Note{}, "", false, err
	}
	return note, string(plain), true, nil
}

func (s *Store) ListNotes() ([]Note, error) {
	rows, err := s.db.Query(`select id, path, title, eyes_only, updated_at from notes order by path collate nocase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []Note
	for rows.Next() {
		var note Note
		var eyesOnly any
		if err := rows.Scan(&note.ID, &note.Path, &note.Title, &eyesOnly, &note.UpdatedAt); err != nil {
			return nil, err
		}
		note.EyesOnly = scanBool(eyesOnly)
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func scanBool(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case []byte:
		return scanBoolString(string(v))
	case string:
		return scanBoolString(v)
	default:
		return false
	}
}

func scanBoolString(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

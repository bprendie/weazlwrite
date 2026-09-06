package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"
)

// SaveDraft writes by note identity so an automatic name can change without
// duplicating the draft. Explicit names never overwrite another note.
func (s *Store) SaveDraft(id, proposed, title, content string, automatic bool) (string, error) {
	if !s.unlocked {
		return "", errors.New("vault is locked")
	}
	proposed = cleanStorePath(proposed)
	if proposed == "" {
		return "", errors.New("vault path is required")
	}
	if err := s.ensureFolderParents(proposed); err != nil {
		return "", err
	}
	nonce, ciphertext, err := s.encrypt([]byte(content))
	if err != nil {
		return "", err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	chosen := proposed
	for n := 2; ; n++ {
		var owner string
		err = tx.QueryRow(`select id from notes where path = ?`, chosen).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) || err == nil && owner == id {
			break
		}
		if err != nil {
			return "", err
		}
		if !automatic {
			return "", errors.New("that vault filename already exists; choose another")
		}
		ext := path.Ext(proposed)
		chosen = fmt.Sprintf("%s-%d%s", strings.TrimSuffix(proposed, ext), n, ext)
	}
	_, err = tx.Exec(`insert into notes (id,path,title,nonce,ciphertext,auto_named)
 values (?,?,?,?,?,?) on conflict(id) do update set path=excluded.path,
 title=excluded.title, nonce=excluded.nonce, ciphertext=excluded.ciphertext,
 auto_named=excluded.auto_named, updated_at=current_timestamp`, id, chosen, title, nonce, ciphertext, automatic)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return chosen, nil
}

func (s *Store) ensureDraftNameColumn() error {
	rows, err := s.db.Query(`pragma table_info(notes)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var value any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &value, &pk); err != nil {
			rows.Close()
			return err
		}
		found = found || name == "auto_named"
	}
	err = rows.Err()
	rows.Close()
	if err != nil || found {
		return err
	}
	_, err = s.db.Exec(`alter table notes add column auto_named integer not null default 0`)
	return err
}

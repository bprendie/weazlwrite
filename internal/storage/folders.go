package storage

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
)

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

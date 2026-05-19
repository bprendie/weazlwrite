package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *model) dropTreeEntry(source, destination treeEntry) error {
	if destination.id == "" {
		return fmt.Errorf("missing destination")
	}
	if !destination.isDir {
		return fmt.Errorf("drop onto a folder")
	}
	if source.vault != destination.vault {
		return fmt.Errorf("cannot move between vault and filesystem; use save to")
	}
	base := entryBaseName(source)
	if base == "" {
		return fmt.Errorf("missing source name")
	}
	if source.vault {
		parent := ""
		if destination.id != "vault:" {
			parent = cleanVaultPath(destination.path)
		}
		next := base
		if parent != "" {
			next = parent + "/" + base
		}
		return m.renameTreeEntry(source, next)
	}
	parent := m.cwd
	if destination.id != "file:" {
		parent = destination.path
	}
	return m.renameTreeEntry(source, filepath.Join(parent, base))
}

func (m *model) renameTreeEntry(entry treeEntry, newPath string) error {
	if entry.id == "" || entry.id == "vault:" || entry.id == "file:" {
		return fmt.Errorf("cannot rename tree root")
	}
	if entry.vault {
		return m.renameVaultTreeEntry(entry, newPath)
	}
	return m.renameFilesystemTreeEntry(entry, newPath)
}

func (m *model) renameVaultTreeEntry(entry treeEntry, newPath string) error {
	clean := cleanVaultPath(newPath)
	if clean == "" {
		return fmt.Errorf("invalid vault path")
	}
	old := cleanVaultPath(entry.path)
	if entry.isDir {
		if err := m.store.RenameFolder(old, clean); err != nil {
			return err
		}
		m.rewriteCurrentVaultPath(old, clean)
		delete(m.treeExpanded, entry.id)
		m.expandTreeTo("vault:" + clean)
		m.status = "moved vault folder:" + clean
		m.err = ""
		return nil
	}
	if err := m.store.RenameNote(old, clean); err != nil {
		return err
	}
	if m.isVault && cleanVaultPath(m.vaultPath) == old {
		m.vaultPath = clean
		m.filePath = clean
	}
	m.expandTreeTo("vault:" + clean)
	m.status = "renamed vault note:" + clean
	m.err = ""
	return nil
}

func (m *model) renameFilesystemTreeEntry(entry treeEntry, newPath string) error {
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	if entry.isDir && isPathInside(abs, entry.path) {
		return fmt.Errorf("cannot move a folder inside itself")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if entry.path == abs {
		return nil
	}
	if err := os.Rename(entry.path, abs); err != nil {
		return err
	}
	m.rewriteCurrentDiskPath(entry.path, abs)
	delete(m.treeExpanded, entry.id)
	m.expandTreeTo("file:" + abs)
	m.status = "moved " + abs
	m.err = ""
	return nil
}

func (m *model) rewriteCurrentVaultPath(oldPath, newPath string) {
	if !m.isVault {
		return
	}
	current := cleanVaultPath(m.vaultPath)
	if current == oldPath || strings.HasPrefix(current, oldPath+"/") {
		m.vaultPath = renamedTreePath(current, oldPath, newPath)
		m.filePath = m.vaultPath
	}
}

func (m *model) rewriteCurrentDiskPath(oldPath, newPath string) {
	if m.isVault || m.diskPath == "" {
		return
	}
	current := filepath.Clean(m.diskPath)
	oldPath = filepath.Clean(oldPath)
	newPath = filepath.Clean(newPath)
	if current == oldPath || strings.HasPrefix(current, oldPath+string(filepath.Separator)) {
		m.diskPath = renamedTreePath(current, oldPath, newPath)
		m.filePath = m.diskPath
		m.cwd = filepath.Dir(m.diskPath)
	}
}

func renamedTreePath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	return newPath + strings.TrimPrefix(path, oldPath)
}

func entryBaseName(entry treeEntry) string {
	if entry.vault {
		return vaultBase(entry.path)
	}
	return filepath.Base(entry.path)
}

func isPathInside(path, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	return path == parent || strings.HasPrefix(path, parent+string(filepath.Separator))
}

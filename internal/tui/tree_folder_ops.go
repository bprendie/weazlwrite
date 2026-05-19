package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (m model) folderBasePath() string {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return ""
	}
	if entry.vault {
		if entry.id == "vault:" {
			return ""
		}
		if entry.isDir {
			return strings.TrimSuffix(entry.path, "/") + "/"
		}
		parent := vaultParent(entry.path)
		if parent == "" {
			return ""
		}
		return parent + "/"
	}
	if entry.id == "file:" {
		return m.cwd + string(filepath.Separator)
	}
	if entry.isDir {
		return entry.path + string(filepath.Separator)
	}
	return filepath.Dir(entry.path) + string(filepath.Separator)
}

func (m *model) createFolder(path string) error {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return fmt.Errorf("no tree selection")
	}
	if entry.vault {
		clean := cleanVaultPath(path)
		if clean == "" {
			return fmt.Errorf("invalid vault folder path")
		}
		if err := m.store.SaveFolder(clean); err != nil {
			return err
		}
		m.expandTreeTo("vault:" + clean)
		m.status = "created vault folder:" + clean
		m.err = ""
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	m.expandTreeTo("file:" + abs)
	m.status = "created folder " + abs
	m.err = ""
	return nil
}

func (m *model) deleteTreeEntry(entry treeEntry) error {
	if entry.id == "" || entry.id == "vault:" || entry.id == "file:" {
		return fmt.Errorf("cannot delete tree root")
	}
	if entry.vault {
		if entry.isDir {
			if err := m.store.DeleteFolder(cleanVaultPath(entry.path)); err != nil {
				return err
			}
			delete(m.treeExpanded, entry.id)
			m.status = "deleted vault folder:" + entry.path
			return nil
		}
		if err := m.store.DeleteNote(entry.path); err != nil {
			return err
		}
		if m.isVault && cleanVaultPath(m.vaultPath) == cleanVaultPath(entry.path) {
			m.vaultID = uuid.NewString()
			m.vaultPath = ""
			m.filePath = ""
			m.dirty = true
		}
		m.status = "deleted vault note:" + entry.path
		return nil
	}
	if entry.path == "" {
		return fmt.Errorf("missing filesystem path")
	}
	if err := os.Remove(entry.path); err != nil {
		return err
	}
	if !m.isVault && m.diskPath == entry.path {
		m.diskPath = ""
		m.filePath = ""
		m.dirty = true
	}
	delete(m.treeExpanded, entry.id)
	m.status = "deleted " + entry.path
	return nil
}

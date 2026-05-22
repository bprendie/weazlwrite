package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (m *model) newVaultNote() {
	name := "untitled-" + uuid.NewString()[:8] + ".md"
	path := m.newVaultNotePath(name)
	m.vaultID = uuid.NewString()
	m.filePath = path
	m.vaultPath = path
	m.diskPath = ""
	m.isVault = true
	m.eyesOnly = false
	m.expandTreeTo("vault:" + path)
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + path
	m.setView(viewEdit)
	m.renderPreview()
}

func (m *model) createTreeDocument() {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		m.err = "select a folder for the new document"
		return
	}
	if entry.vault {
		if err := m.createVaultDocumentAtSelection(entry); err != nil {
			m.err = err.Error()
		}
		return
	}
	if err := m.createFilesystemDocumentAtSelection(entry); err != nil {
		m.err = err.Error()
	}
}

func (m *model) createVaultDocumentAtSelection(entry treeEntry) error {
	name := "untitled-" + uuid.NewString()[:8] + ".md"
	path := cleanVaultPath(m.vaultDocumentBase(entry) + name)
	if path == "" {
		return fmt.Errorf("invalid vault path")
	}
	content := "# Untitled\n\n"
	id := uuid.NewString()
	if err := m.store.SaveNote(id, path, titleFor(path, content), content); err != nil {
		return err
	}
	m.vaultID = id
	m.filePath = path
	m.vaultPath = path
	m.diskPath = ""
	m.isVault = true
	m.eyesOnly = false
	m.editor.SetValue(content)
	m.dirty = false
	m.err = ""
	m.status = "created vault note " + path
	m.expandTreeTo("vault:" + path)
	if err := m.renderTree(); err != nil {
		return err
	}
	m.setView(viewEdit)
	m.renderPreview()
	return nil
}

func (m *model) createFilesystemDocumentAtSelection(entry treeEntry) error {
	dir := m.filesystemDocumentDir(entry)
	if dir == "" {
		return fmt.Errorf("select a filesystem folder")
	}
	path := filepath.Join(dir, "untitled-"+uuid.NewString()[:8]+".md")
	content := "# Untitled\n\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	m.filePath = path
	m.diskPath = path
	m.vaultPath = ""
	m.vaultID = ""
	m.cwd = filepath.Dir(path)
	m.isVault = false
	m.eyesOnly = false
	m.editor.SetValue(content)
	m.dirty = false
	m.err = ""
	m.status = "created " + path
	m.expandTreeTo("file:" + path)
	if err := m.renderTree(); err != nil {
		return err
	}
	m.setView(viewEdit)
	m.renderPreview()
	return nil
}

func (m model) newVaultNotePath(name string) string {
	base := m.selectedVaultFolderBase()
	if base == "" && m.isVault && m.vaultPath != "" {
		if parent := vaultParent(m.vaultPath); parent != "" {
			base = parent + "/"
		}
	}
	return cleanVaultPath(base + name)
}

func (m model) vaultDocumentBase(entry treeEntry) string {
	if !entry.vault || entry.id == "vault:" {
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

func (m model) filesystemDocumentDir(entry treeEntry) string {
	if entry.vault {
		return ""
	}
	if entry.id == "file:" {
		return m.cwd
	}
	if entry.isDir {
		return entry.path
	}
	if entry.path != "" {
		return filepath.Dir(entry.path)
	}
	return ""
}

func (m model) selectedVaultFolderBase() string {
	entry, ok := m.selectedTreeEntry()
	if !ok || !entry.vault {
		return ""
	}
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

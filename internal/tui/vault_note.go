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
	m.loadEditorText("# Untitled\n")
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
	path := m.documentBasePath(entry) + "untitled-" + uuid.NewString()[:8] + ".md"
	if err := m.createTreeDocumentAtPath(entry, path); err != nil {
		m.err = err.Error()
	}
}

func (m *model) createTreeDocumentAtPath(entry treeEntry, path string) error {
	if entry.vault {
		return m.createVaultDocumentAtPath(path)
	}
	return m.createFilesystemDocumentAtPath(path)
}

func (m *model) createVaultDocumentAtSelection(entry treeEntry) error {
	path := m.vaultDocumentBase(entry) + "untitled-" + uuid.NewString()[:8] + ".md"
	return m.createVaultDocumentAtPath(path)
}

func (m *model) createVaultDocumentAtPath(path string) error {
	path = cleanVaultDocumentPath(path)
	if path == "" {
		return fmt.Errorf("invalid vault path")
	}
	content := "# Untitled\n"
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
	m.loadEditorText(content)
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
	return m.createFilesystemDocumentAtPath(filepath.Join(dir, "untitled-"+uuid.NewString()[:8]+".md"))
}

func (m *model) createFilesystemDocumentAtPath(path string) error {
	path = cleanFilesystemDocumentPath(path)
	if path == "" {
		return fmt.Errorf("invalid filesystem path")
	}
	content := "# Untitled\n"
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
	m.loadEditorText(content)
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

func (m model) documentBasePath(entry treeEntry) string {
	if entry.vault {
		return m.vaultDocumentBase(entry)
	}
	dir := m.filesystemDocumentDir(entry)
	if dir == "" {
		return ""
	}
	return dir + string(filepath.Separator)
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

func cleanVaultDocumentPath(path string) string {
	clean := cleanVaultPath(path)
	if clean == "" || strings.HasSuffix(clean, "/") {
		return ""
	}
	if filepath.Ext(clean) == "" {
		clean += ".md"
	}
	return clean
}

func cleanFilesystemDocumentPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasSuffix(path, string(filepath.Separator)) || strings.HasSuffix(path, "/") {
		return ""
	}
	if filepath.Ext(path) == "" {
		path += ".md"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return abs
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

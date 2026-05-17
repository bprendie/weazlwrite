package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (m *model) openDiskPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err == nil && info.IsDir() {
		m.cwd = abs
		return m.renderTree()
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if isConvertibleDocument(abs) {
		return m.importAndOpenDocument(abs)
	}
	b, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		b = []byte("# " + strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)) + "\n\n")
	} else if err != nil {
		return err
	}
	m.filePath = abs
	m.diskPath = abs
	m.vaultPath = ""
	m.cwd = filepath.Dir(abs)
	m.isVault = false
	m.eyesOnly = false
	m.vaultID = ""
	m.expandTreeTo("file:" + abs)
	m.editor.SetValue(string(b))
	m.dirty = false
	m.status = "editing " + abs
	m.err = ""
	m.setView(viewEdit)
	if err := m.renderTree(); err != nil {
		return err
	}
	m.renderPreview()
	return nil
}

func (m *model) importAndOpenDocument(path string) error {
	target := importPathForFile(path, m.cwd)
	if err := m.importFileToVault(path, target); err != nil {
		return err
	}
	if err := m.renderTree(); err != nil {
		return err
	}
	if err := m.openVaultPath(target); err != nil {
		return err
	}
	m.status = "imported and opened vault:" + target
	return nil
}

func (m *model) openVaultPath(path string) error {
	note, content, ok, err := m.store.LoadNote(path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("vault note not found: %s", path)
	}
	m.filePath = note.Path
	m.vaultPath = note.Path
	m.diskPath = ""
	m.vaultID = note.ID
	m.isVault = true
	m.eyesOnly = note.EyesOnly
	if m.eyesOnly {
		m.mouseCapture = true
	}
	m.expandTreeTo("vault:" + note.Path)
	m.editor.SetValue(content)
	m.dirty = false
	m.status = "editing vault:" + note.Path
	if m.eyesOnly {
		m.status = "eyes only vault:" + note.Path
	}
	m.err = ""
	m.setView(viewEdit)
	m.renderPreview()
	return nil
}

func (m *model) save() {
	if m.isVault {
		m.saveToVault()
		return
	}
	if m.diskPath == "" {
		m.err = "no filesystem path; use ctrl+f"
		return
	}
	if err := m.saveToDiskPath(m.diskPath); err != nil {
		m.err = err.Error()
		return
	}
}

func (m *model) saveToVault() {
	path := m.vaultPath
	if path == "" {
		path = defaultVaultPath(m.diskPath, m.cwd, m.editor.Value())
	}
	if err := m.saveToVaultPath(path); err != nil {
		m.err = err.Error()
	}
}

func (m *model) saveToVaultPath(path string) error {
	content := m.editor.Value()
	if m.vaultID == "" {
		m.vaultID = uuid.NewString()
	}
	path = cleanVaultPath(path)
	if path == "" {
		return fmt.Errorf("invalid vault path")
	}
	if err := m.store.SaveNote(m.vaultID, path, titleFor(path, content), content); err != nil {
		return err
	}
	m.vaultPath = path
	m.filePath = path
	m.isVault = true
	m.eyesOnly = m.eyesOnlyPaths[cleanVaultPath(path)]
	m.expandTreeTo("vault:" + path)
	m.status = "saved vault:" + path
	m.dirty = false
	m.err = ""
	return nil
}

func (m *model) saveToDiskPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, []byte(m.editor.Value()), 0o644); err != nil {
		return err
	}
	m.filePath = abs
	m.diskPath = abs
	m.isVault = false
	m.cwd = filepath.Dir(abs)
	m.expandTreeTo("file:" + abs)
	m.status = "saved " + abs
	m.dirty = false
	m.err = ""
	return nil
}

func defaultVaultPath(diskPath, cwd, content string) string {
	if diskPath != "" {
		if rel, err := filepath.Rel(cwd, diskPath); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			return cleanVaultPath(rel)
		}
		return cleanVaultPath(filepath.Base(diskPath))
	}
	title := strings.ToLower(strings.ReplaceAll(titleFor("", content), " ", "-"))
	if title == "" {
		title = "untitled"
	}
	if filepath.Ext(title) == "" {
		title += ".md"
	}
	return cleanVaultPath(title)
}

package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlwrite/internal/importer"
)

func (m model) startImportSelectedToVault() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if entry.vault {
		m.err = "select a filesystem file or folder to import"
		return m, nil
	}
	if entry.id == "file:" {
		m.err = "select a filesystem file or folder to import"
		return m, nil
	}
	m.mode = modeImporting
	m.aiBusy = true
	m.generatingAt = time.Now()
	m.err = ""
	m.status = "importing " + entry.path
	m.editor.Blur()
	return m, tea.Batch(m.importFilesystemEntryCmd(entry), m.working.Tick)
}

func (m model) importFilesystemEntryCmd(entry treeEntry) tea.Cmd {
	return func() tea.Msg {
		files, folders, warnings, err := m.importFilesystemEntry(entry)
		return importResultMsg{files: files, folders: folders, warnings: warnings, err: err}
	}
}

func (m *model) importFilesystemEntry(entry treeEntry) (int, int, int, error) {
	info, err := os.Stat(entry.path)
	if err != nil {
		return 0, 0, 0, err
	}
	if !info.IsDir() {
		if !isVaultImportFile(entry.path) {
			return 0, 0, 0, fmt.Errorf("only markdown, text, PDF, and DOCX files can be imported")
		}
		target := importPathForFile(entry.path, m.cwd)
		if err := m.importFileToVault(entry.path, target); err != nil {
			return 0, 0, 0, err
		}
		m.expandTreeTo("vault:" + target)
		return 1, 0, 0, nil
	}

	root := entry.path
	files := 0
	folders := 0
	warnings := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := cleanVaultPath(filepath.ToSlash(rel))
		if target == "" {
			return nil
		}
		if d.IsDir() {
			if err := m.store.SaveFolder(target); err != nil {
				return err
			}
			folders++
			return nil
		}
		if !isVaultImportFile(path) {
			return nil
		}
		if err := m.importFileToVault(path, target); err != nil {
			if errors.Is(err, importer.ErrImageOnlyDocument) {
				warnings++
				return nil
			}
			return err
		}
		files++
		return nil
	})
	if err != nil {
		return files, folders, warnings, err
	}
	m.expandTreeTo("vault:")
	return files, folders, warnings, nil
}

func (m *model) importFileToVault(sourcePath, vaultPath string) error {
	doc, err := importer.Convert(sourcePath)
	if err != nil {
		return err
	}
	return m.store.SaveNote(uuid.NewString(), vaultPath, titleFor(vaultPath, doc.Markdown), doc.Markdown)
}

func isVaultImportFile(path string) bool {
	return importer.Supported(path) || strings.EqualFold(filepath.Ext(path), ".doc")
}

func importPathForFile(path, cwd string) string {
	path = importer.MarkdownPath(path)
	rel, err := filepath.Rel(cwd, path)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		if clean := cleanVaultPath(filepath.ToSlash(rel)); clean != "" {
			return clean
		}
	}
	return cleanVaultPath(filepath.Base(path))
}

func isConvertibleDocument(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf", ".docx":
		return true
	default:
		return false
	}
}

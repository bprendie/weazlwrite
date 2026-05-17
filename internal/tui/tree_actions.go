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

func (m model) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.treeIdx > 0 {
			m.treeIdx--
		}
	case "down", "j":
		if m.treeIdx < len(m.tree)-1 {
			m.treeIdx++
		}
	case "pgup":
		m.pageTree(-1)
	case "pgdown":
		m.pageTree(1)
	case "enter":
		return m, m.openSelected()
	case " ":
		m.pickupOrDropSelected()
	case "n":
		return m.startNewFolder()
	case "d":
		return m.startConfirmDelete()
	case "r":
		return m.startRenameTree()
	case "i":
		return m.startImportSelectedToVault()
	case "o":
		return m.toggleEyesOnlySelected()
	case "esc":
		m.carryTarget = treeEntry{}
		m.setMainFocus()
	}
	m.ensureTreeSelectionVisible()
	return m, nil
}

func (m *model) pageTree(direction int) {
	if len(m.tree) == 0 {
		return
	}
	height := max(1, m.treeContentHeight())
	step := max(1, height-1)
	m.treeIdx = min(max(0, m.treeIdx+direction*step), len(m.tree)-1)
	m.ensureTreeSelectionVisible()
}

func (m *model) scrollTree(delta int) {
	if len(m.tree) == 0 {
		return
	}
	height := max(1, m.treeContentHeight())
	maxOffset := max(0, len(m.tree)-height)
	m.treeOffset = min(max(0, m.treeOffset+delta), maxOffset)
	if m.treeIdx < m.treeOffset {
		m.treeIdx = m.treeOffset
	}
	if m.treeIdx >= m.treeOffset+height {
		m.treeIdx = min(len(m.tree)-1, m.treeOffset+height-1)
	}
}

func (m *model) afterUnlock() error {
	if err := m.renderTree(); err != nil {
		return err
	}
	if m.filePath != "" {
		return m.openDiskPath(m.filePath)
	}
	m.newVaultNote()
	return nil
}

func (m *model) newVaultNote() {
	name := "untitled-" + uuid.NewString()[:8] + ".md"
	m.vaultID = uuid.NewString()
	m.filePath = name
	m.vaultPath = name
	m.diskPath = ""
	m.isVault = true
	m.eyesOnly = false
	m.expandTreeTo("vault:" + name)
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + name
	m.setView(viewEdit)
	m.renderPreview()
}

func (m *model) openSelected() tea.Cmd {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return nil
	}
	entry := m.tree[m.treeIdx]
	if entry.isDir {
		m.toggleTreeEntry(entry)
		return nil
	}
	if entry.vault && !entry.isDir {
		if err := m.openVaultPath(entry.path); err != nil {
			m.err = err.Error()
		}
		if m.eyesOnly {
			return tea.EnableMouseCellMotion
		}
		return nil
	}
	if !entry.isDir {
		if err := m.openDiskPath(entry.path); err != nil {
			m.err = err.Error()
		}
	}
	return nil
}

func (m *model) toggleSelectedDir() {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return
	}
	entry := m.tree[m.treeIdx]
	if entry.isDir {
		m.toggleTreeEntry(entry)
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) toggleTreeEntry(entry treeEntry) {
	if entry.id == "" {
		return
	}
	m.treeExpanded[entry.id] = !m.treeExpanded[entry.id]
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) pickupOrDropSelected() {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return
	}
	if m.carryTarget.id == "" {
		if entry.id == "vault:" || entry.id == "file:" || entry.isDir {
			m.err = "space picks up files only; press enter to fold folders"
			return
		}
		m.carryTarget = entry
		m.err = ""
		m.status = "picked up " + entry.path
		return
	}
	source := m.carryTarget
	if err := m.dropTreeEntry(source, entry); err != nil {
		m.err = err.Error()
		return
	}
	m.carryTarget = treeEntry{}
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
}

func (m model) selectedTreeEntry() (treeEntry, bool) {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return treeEntry{}, false
	}
	return m.tree[m.treeIdx], true
}

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

func (m model) toggleEyesOnlySelected() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if !entry.vault || entry.isDir || entry.id == "vault:" {
		m.err = "eyes only is available for vault files"
		return m, nil
	}
	return m.toggleEyesOnlyPath(cleanVaultPath(entry.path), entry)
}

func (m model) toggleCurrentEyesOnly() (tea.Model, tea.Cmd) {
	if !m.isVault || strings.TrimSpace(m.vaultPath) == "" {
		m.err = "open a vault note to toggle eyes only"
		return m, nil
	}
	path := cleanVaultPath(m.vaultPath)
	return m.toggleEyesOnlyPath(path, treeEntry{
		id:    "vault:" + path,
		name:  filepath.Base(path),
		path:  path,
		vault: true,
	})
}

func (m model) toggleEyesOnlyPath(path string, entry treeEntry) (tea.Model, tea.Cmd) {
	if path == "" {
		m.err = "missing vault note path"
		return m, nil
	}
	next := !m.eyesOnlyPaths[path]
	if !next {
		entry.path = path
		entry.vault = true
		m.eyesOffTarget = entry
		m.mode = modeConfirmEyesOff
		m.status = "confirm disabling eyes only"
		m.err = ""
		return m, nil
	}
	if err := m.store.SetNoteEyesOnly(path, true); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if m.eyesOnlyPaths == nil {
		m.eyesOnlyPaths = map[string]bool{}
	}
	m.eyesOnlyPaths[path] = true
	if m.isVault && cleanVaultPath(m.vaultPath) == path {
		m.eyesOnly = true
		m.mouseCapture = true
	}
	m.status = "eyes only enabled:" + path
	m.err = ""
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
	if m.isVault && cleanVaultPath(m.vaultPath) == path {
		return m, tea.EnableMouseCellMotion
	}
	return m, nil
}

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
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	if entry.isDir {
		if isPathInside(abs, entry.path) {
			return fmt.Errorf("cannot move a folder inside itself")
		}
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

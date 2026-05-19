package tui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	case "left":
		m.collapseSelectedDir()
	case "right":
		m.expandSelectedDir()
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

func (m *model) expandSelectedDir() {
	m.setSelectedDirExpanded(true)
}

func (m *model) collapseSelectedDir() {
	m.setSelectedDirExpanded(false)
}

func (m *model) setSelectedDirExpanded(expanded bool) {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return
	}
	entry := m.tree[m.treeIdx]
	if !entry.isDir || entry.id == "" {
		return
	}
	if m.treeExpanded[entry.id] == expanded {
		return
	}
	m.setTreeEntryExpanded(entry, expanded)
}

func (m *model) toggleTreeEntry(entry treeEntry) {
	if entry.id == "" {
		return
	}
	m.setTreeEntryExpanded(entry, !m.treeExpanded[entry.id])
}

func (m *model) setTreeEntryExpanded(entry treeEntry, expanded bool) {
	m.treeExpanded[entry.id] = expanded
	if entry.vault && entry.isDir && entry.id != "vault:" {
		if err := m.persistVaultFolderExpanded(entry.path, m.treeExpanded[entry.id]); err != nil {
			m.err = err.Error()
		}
	}
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

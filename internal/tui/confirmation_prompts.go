package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startConfirmDelete() (tea.Model, tea.Cmd) {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return m, nil
	}
	entry := m.tree[m.treeIdx]
	if entry.id == "vault:" || entry.id == "file:" {
		m.err = "cannot delete tree root"
		return m, nil
	}
	m.deleteTarget = entry
	m.mode = modeConfirmDelete
	m.status = "confirm delete"
	return m, nil
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if err := m.deleteTreeEntry(m.deleteTarget); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.deleteTarget = treeEntry{}
		m.renderTree()
		return m, nil
	case "n", "N", "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.deleteTarget = treeEntry{}
		m.status = "delete cancelled"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateConfirmEyesOff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		entry := m.eyesOffTarget
		path := cleanVaultPath(entry.path)
		if err := m.store.SetNoteEyesOnly(path, false); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		if m.eyesOnlyPaths != nil {
			delete(m.eyesOnlyPaths, path)
		}
		if m.isVault && cleanVaultPath(m.vaultPath) == path {
			m.eyesOnly = false
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.eyesOffTarget = treeEntry{}
		m.err = ""
		m.status = "eyes only disabled:" + path
		if err := m.renderTree(); err != nil {
			m.err = err.Error()
		}
		return m, nil
	case "n", "N", "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.eyesOffTarget = treeEntry{}
		m.status = "eyes only unchanged"
		return m, nil
	default:
		return m, nil
	}
}

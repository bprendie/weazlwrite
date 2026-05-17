package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startSaveFile() (tea.Model, tea.Cmd) {
	path := m.diskPath
	if path == "" {
		name := m.vaultPath
		if name == "" {
			name = m.filePath
		}
		if name == "" {
			name = strings.ToLower(strings.ReplaceAll(titleFor("", m.editor.Value()), " ", "-")) + ".md"
		}
		path = filepath.Join(m.cwd, filepath.Base(name))
	}
	m.mode = modeSaveFile
	m.filePrompt.SetValue(path)
	m.filePrompt.Focus()
	m.editor.Blur()
	m.status = "save to: filesystem"
	return m, textinput.Blink
}

func (m model) startSaveVault() (tea.Model, tea.Cmd) {
	path := m.vaultPath
	if path == "" {
		path = defaultVaultPath(m.diskPath, m.cwd, m.editor.Value())
	}
	m.mode = modeSaveVault
	m.vaultPrompt.SetValue(path)
	m.vaultPrompt.Focus()
	m.editor.Blur()
	m.status = "save to: encrypted vault"
	return m, textinput.Blink
}

func (m model) updateSaveFile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.filePrompt.Value())
		if path == "" {
			m.err = "filesystem path is required"
			return m, nil
		}
		if err := m.saveToDiskPath(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "filesystem save cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.filePrompt, cmd = m.filePrompt.Update(msg)
		return m, cmd
	}
}

func (m model) updateSaveVault(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.vaultPrompt.Value())
		if path == "" {
			m.err = "vault path is required"
			return m, nil
		}
		if err := m.saveToVaultPath(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "vault save cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.vaultPrompt, cmd = m.vaultPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) startNewFolder() (tea.Model, tea.Cmd) {
	if m.focus != focusTree {
		m.focus = focusTree
	}
	base := m.folderBasePath()
	m.mode = modeNewFolder
	m.folderPrompt.SetValue(base)
	m.folderPrompt.Focus()
	m.editor.Blur()
	m.status = "new folder"
	return m, textinput.Blink
}

func (m model) updateNewFolder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.folderPrompt.Value())
		if path == "" {
			m.err = "folder path is required"
			return m, nil
		}
		if err := m.createFolder(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.status = "new folder cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.folderPrompt, cmd = m.folderPrompt.Update(msg)
		return m, cmd
	}
}

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

func (m model) startRenameTree() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if entry.id == "vault:" || entry.id == "file:" {
		m.err = "cannot rename tree root"
		return m, nil
	}
	m.renameTarget = entry
	m.mode = modeRenameTree
	m.renamePrompt.SetValue(entry.path)
	m.renamePrompt.Focus()
	m.editor.Blur()
	m.status = "rename/move"
	return m, textinput.Blink
}

func (m model) updateRenameTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.renamePrompt.Value())
		if path == "" {
			m.err = "path is required"
			return m, nil
		}
		if err := m.renameTreeEntry(m.renameTarget, path); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.renameTarget = treeEntry{}
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.renameTarget = treeEntry{}
		m.status = "rename cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.renamePrompt, cmd = m.renamePrompt.Update(msg)
		return m, cmd
	}
}

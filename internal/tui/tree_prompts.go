package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

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

func (m model) startNewDocument() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		m.err = "select a folder for the new document"
		return m, nil
	}
	base := m.documentBasePath(entry)
	m.newDocTarget = entry
	m.mode = modeNewDocument
	m.renamePrompt.SetValue(base)
	m.renamePrompt.Focus()
	m.editor.Blur()
	m.status = "new document"
	return m, textinput.Blink
}

func (m model) updateNewDocument(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.renamePrompt.Value())
		if path == "" {
			m.err = "document path is required"
			return m, nil
		}
		if err := m.createTreeDocumentAtPath(m.newDocTarget, path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.newDocTarget = treeEntry{}
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.newDocTarget = treeEntry{}
		m.status = "new document cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.renamePrompt, cmd = m.renamePrompt.Update(msg)
		return m, cmd
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

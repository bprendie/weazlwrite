package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"path/filepath"
	"strings"
)

func (m model) startSaveFile() (tea.Model, tea.Cmd) {
	path := m.diskPath
	if path == "" {
		name := m.vaultPath
		if name == "" {
			name = m.filePath
		}
		if name == "" {
			name = strings.ToLower(strings.ReplaceAll(titleFor("", m.editorText()), " ", "-")) + ".md"
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
	if m.autoNamed {
		path = m.draftNameSuggestion()
	}
	if path == "" {
		path = defaultVaultPath(m.diskPath, m.cwd, m.editorText())
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
		var err error
		if m.autoNamed {
			err = m.acceptDraftName(path)
		} else {
			err = m.saveToVaultPath(path)
		}
		if err != nil {
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

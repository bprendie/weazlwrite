package tui

import tea "github.com/charmbracelet/bubbletea"

type pendingDiskTransition struct {
	key   tea.KeyMsg
	mode  mode
	focus focus
}

func (m model) leavesDiskDraft(key tea.KeyMsg) bool {
	if key.Paste {
		return false
	}
	k := key.String()
	if (k == "ctrl+c" && !m.editorTyping()) || k == "ctrl+q" {
		return true
	}
	if m.mode == modeWrite {
		if k == keyNewNote {
			return true
		}
		if m.focus == focusTree {
			if k == "n" || k == "i" {
				return true
			}
			if k == "enter" {
				entry, ok := m.selectedTreeEntry()
				return ok && !entry.isDir
			}
		}
	}
	if m.mode == modeNewDocument && k == "enter" {
		return true
	}
	if m.mode == modeConfirmDelete && (k == "enter" || k == "y" || k == "Y") {
		return !m.deleteTarget.vault && m.deleteTarget.path == m.diskPath
	}
	return false
}

func (m model) confirmUnsavedDisk(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.diskTransition = &pendingDiskTransition{key: key, mode: m.mode, focus: m.focus}
	m.mode = modeConfirmUnsaved
	m.editor.Blur()
	m.undoOpen = false
	m.err = ""
	m.status = "unsaved draft"
	return m, nil
}

func (m model) updateUnsavedDisk(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Paste || m.diskTransition == nil {
		return m, nil
	}
	pending := *m.diskTransition
	switch key.String() {
	case "esc", "c":
		m.diskTransition = nil
		m.mode = pending.mode
		m.setFocus(pending.focus)
		m.err = ""
		m.status = "staying with current draft"
		return m, nil
	case "s", "ctrl+s":
		if m.isVault {
			m.diskTransition = nil
			m.mode = pending.mode
			m.setFocus(pending.focus)
			m.saves.pending = pending.key
			cmd := m.startVaultSave(true)
			return m, cmd
		}
		m.save()
		if m.err != "" {
			return m, nil
		}
	case "d":
		m.dirty = false
	default:
		return m, nil
	}
	m.diskTransition = nil
	m.mode = pending.mode
	m.setFocus(pending.focus)
	m.err = ""
	return m.updateAndSchedule(pending.key)
}

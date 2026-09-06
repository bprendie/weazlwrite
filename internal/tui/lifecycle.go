package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

func (m *model) enforceAutoLock() bool {
	if m.saves.inFlight != nil || m.aiBusy || m.store == nil || !m.store.AutoLockExpired() {
		return false
	}
	if m.dirty {
		m.save()
		if m.err != "" {
			m.store.UpdateActivity()
			m.status = "auto-lock delayed until current document saves"
			return false
		}
	}
	m.store.Lock()
	m.applyAutoLock()
	return true
}

func autoLockTick() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return autoLockTickMsg{}
	})
}

func (m *model) recordActivity() {
	if m.store != nil && m.store.Unlocked() {
		m.store.UpdateActivity()
	}
}

func (m *model) applyAutoLock() {
	m.diskTransition = nil
	m.mode = modeVault
	m.focus = focusEditor
	m.password.SetValue("")
	m.password.Focus()
	m.pendingPass = ""
	m.confirmPass.SetValue("")
	m.loadEditorText("")
	m.preview.SetContent("")
	m.tree = nil
	m.treeIdx = 0
	m.treeOffset = 0
	m.eyesOnlyPaths = map[string]bool{}
	m.filePath = ""
	m.diskPath = ""
	m.vaultPath = ""
	m.vaultID = ""
	m.isVault = false
	m.eyesOnly = false
	m.selectionMode = false
	m.selecting = false
	m.editorDrag = false
	m.mouseCapture = true
	m.dirty = false
	m.prepareVaultPassword()
	m.status = "vault auto-locked"
	m.err = "vault auto-locked after inactivity"
}

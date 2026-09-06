package tui

import (
	"fmt"
	"time"

	"github.com/bprendie/weazlwrite/internal/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

const (
	vaultSavePause    = 750 * time.Millisecond
	vaultSaveInterval = 5 * time.Second
)

// Only the Tea loop owns this state. Commands receive immutable snapshots.
type vaultSaveState struct {
	version    uint64
	timer      uint64
	armed      bool
	lastEdit   time.Time
	firstDirty time.Time
	inFlight   *vaultSnapshot
	flush      bool
	confirm    bool
	pending    tea.Msg
	err        string
}

type vaultSnapshot struct {
	store          *storage.Store
	epoch, version uint64
	id, path, text string
	automatic      bool
}

type vaultSaveResult struct {
	snapshot *vaultSnapshot
	path     string
	err      error
}
type vaultSaveWake struct {
	epoch, timer uint64
	at           time.Time
}
type lockVaultMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vaultSaveResult:
		return m.finishVaultSave(msg)
	case vaultSaveWake:
		if msg.epoch != m.editorEpoch || msg.timer != m.saves.timer || !m.saves.armed {
			return m, nil
		}
		m.saves.armed = false
		if !m.dirty || !m.isVault {
			return m, nil
		}
		if !msg.at.Before(m.nextVaultSave()) {
			cmd := m.startVaultSave(false)
			return m, cmd
		}
		cmd := m.armVaultSave(msg.at)
		return m, cmd
	}
	if m.mode == modeConfirmUnsaved {
		if key, ok := msg.(tea.KeyMsg); ok {
			return m.updateUnsavedDisk(key)
		}
		if _, mouse := msg.(tea.MouseMsg); mouse {
			return m, nil
		}
	}
	if m.saves.pending != nil {
		// Do not let a second command change the target of a pending transition.
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.saves.pending = nil
				m.status = "transition cancelled; saving continues"
			}
			return m, nil
		case tea.MouseMsg:
			return m, nil
		}
	}
	if m.store != nil && m.store.AutoLockExpired() && !m.aiBusy {
		switch msg.(type) {
		case tea.KeyMsg, tea.MouseMsg, autoLockTickMsg:
			if m.isVault && m.dirty && m.vaultPath == "" {
				// A deleted note may leave an unattached buffer. Saving it under
				// a new name requires intent, just as leaving that buffer does.
				m.store.UpdateActivity()
				m.saves.err = "auto-lock delayed: draft has no vault path; Ctrl+S saves"
			} else if m.isVault && (m.dirty || m.saves.inFlight != nil) {
				m.saves.pending = lockVaultMsg{}
				cmd := m.startVaultSave(true)
				if _, tick := msg.(autoLockTickMsg); tick {
					cmd = tea.Batch(cmd, autoLockTick())
				}
				return m, cmd
			} else if m.enforceAutoLock() {
				cmd := textinput.Blink
				if _, tick := msg.(autoLockTickMsg); tick {
					cmd = tea.Batch(cmd, autoLockTick())
				}
				return m, cmd
			}
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		m.recordActivity()
		if (!m.isVault || m.vaultPath == "") && m.dirty && m.leavesDiskDraft(key) {
			return m.confirmUnsavedDisk(key)
		}
		if key.String() == keySave && m.mode == modeWrite && m.isVault {
			m.undoOpen = false
			if m.autoNamed {
				return m.firstVaultSave(key)
			}
			m.saves.confirm = true
			cmd := m.startVaultSave(true)
			return m, cmd
		}
		if m.vaultTransition(key) && m.isVault && m.vaultPath != "" && (m.dirty || m.saves.inFlight != nil) {
			m.saves.pending = msg
			cmd := m.startVaultSave(true)
			return m, cmd
		}
	}
	return m.updateAndSchedule(msg)
}

func (m model) updateAndSchedule(msg tea.Msg) (tea.Model, tea.Cmd) {
	before, epoch := m.editorText(), m.editorEpoch
	updated, cmd := m.update(msg)
	next := updated.(model)
	if next.dirty && (next.editorEpoch != epoch || next.editorText() != before) {
		next.saves.version++
		next.saves.lastEdit = time.Now()
		if next.saves.firstDirty.IsZero() {
			next.saves.firstDirty = next.saves.lastEdit
		}
		if next.isVault && next.vaultPath != "" {
			cmd = tea.Batch(cmd, next.armVaultSave(next.saves.lastEdit))
		}
	}
	return next, cmd
}

func (m model) nextVaultSave() time.Time {
	pause := m.saves.lastEdit.Add(vaultSavePause)
	limit := m.saves.firstDirty.Add(vaultSaveInterval)
	if limit.Before(pause) {
		return limit
	}
	return pause
}

func (m *model) armVaultSave(now time.Time) tea.Cmd {
	if m.saves.armed || m.saves.inFlight != nil || m.saves.err != "" || !m.isVault || !m.dirty {
		return nil
	}
	m.saves.timer++
	m.saves.armed = true
	epoch, timer := m.editorEpoch, m.saves.timer
	delay := m.nextVaultSave().Sub(now)
	if delay < time.Millisecond {
		delay = time.Millisecond
	}
	return tea.Tick(delay, func(at time.Time) tea.Msg {
		return vaultSaveWake{epoch: epoch, timer: timer, at: at}
	})
}

func (m *model) startVaultSave(flush bool) tea.Cmd {
	m.saves.flush = m.saves.flush || flush
	m.saves.armed = false
	m.saves.timer++
	if m.saves.inFlight != nil {
		return nil
	}
	if !m.dirty {
		m.saves.flush = false
		m.confirmVaultSave(m.vaultPath)
		return nil
	}
	if m.store == nil || !m.store.Unlocked() {
		m.saves.err = "vault is locked"
		m.saves.pending = nil
		return nil
	}
	if m.vaultPath == "" {
		m.vaultPath = defaultVaultPath(m.diskPath, m.cwd, m.editorText())
		m.filePath = m.vaultPath
	}
	if m.vaultID == "" {
		m.vaultID = uuid.NewString()
	}
	snapshot := &vaultSnapshot{store: m.store, epoch: m.editorEpoch, version: m.saves.version, id: m.vaultID, path: m.vaultPath, text: m.editorText(), automatic: m.autoNamed}
	m.saves.inFlight = snapshot
	m.saves.err = ""
	return func() tea.Msg {
		if snapshot.automatic {
			path, err := snapshot.store.SaveDraft(snapshot.id, autoDraftPath(snapshot.path, snapshot.text), titleFor("", snapshot.text), snapshot.text, true)
			return vaultSaveResult{snapshot: snapshot, path: path, err: err}
		}
		err := snapshot.store.SaveNote(snapshot.id, snapshot.path, titleFor(snapshot.path, snapshot.text), snapshot.text)
		return vaultSaveResult{snapshot: snapshot, err: err}
	}
}

func (m model) finishVaultSave(msg vaultSaveResult) (tea.Model, tea.Cmd) {
	if msg.snapshot == nil || m.saves.inFlight != msg.snapshot {
		return m, nil
	}
	m.saves.inFlight = nil
	snapshot := msg.snapshot
	if snapshot.store != m.store || snapshot.epoch != m.editorEpoch || snapshot.path != m.vaultPath || snapshot.id != m.vaultID {
		// Transitions normally drain writes first; never trust stale completions.
		return m, nil
	}
	if msg.err != nil {
		if _, locking := m.saves.pending.(lockVaultMsg); locking {
			m.store.UpdateActivity()
		}
		m.saves.err = fmt.Sprintf("save failed: %v; Ctrl+S retries", msg.err)
		m.saves.flush = false
		m.saves.pending = nil
		m.dirty = true
		return m, nil
	}
	if msg.path != "" {
		m.vaultPath, m.filePath = msg.path, msg.path
		m.expandTreeTo("vault:" + msg.path)
	}
	if snapshot.version == m.saves.version {
		m.dirty = false
		m.saves.firstDirty = time.Time{}
		m.saves.flush = false
		m.saves.err = ""
		m.confirmVaultSave(m.vaultPath)
		if err := m.renderTree(); err != nil {
			m.err = err.Error()
		}
		if pending := m.saves.pending; pending != nil {
			m.saves.pending = nil
			if _, lock := pending.(lockVaultMsg); lock {
				m.store.Lock()
				m.applyAutoLock()
				return m, nil
			}
			return m.Update(pending)
		}
		return m, nil
	}
	// Edits made during a slow write remain dirty and get the next snapshot.
	if m.saves.flush || time.Since(m.saves.firstDirty) >= vaultSaveInterval {
		cmd := m.startVaultSave(false)
		return m, cmd
	}
	cmd := m.armVaultSave(time.Now())
	return m, cmd
}

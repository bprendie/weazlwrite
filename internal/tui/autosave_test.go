package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bprendie/weazlwrite/internal/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func vaultModel(t *testing.T) model {
	t.Helper()
	m := inputModel()
	m.password = textinput.New()
	m.confirmPass = textinput.New()
	m.cwd = t.TempDir()
	store, err := storage.Open(filepath.Join(m.cwd, "vault.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err = store.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err = store.CreateVault("disposable-password"); err != nil {
		t.Fatal(err)
	}
	m.store = store
	m.treeExpanded = map[string]bool{"vault:": true, "file:": true}
	m.isVault, m.vaultPath, m.filePath, m.vaultID = true, "draft.md", "draft.md", "draft-id"
	m.width, m.height = 100, 30
	m.resize()
	return m
}

func updateModel(m model, msg tea.Msg) (model, tea.Cmd) {
	u, cmd := m.Update(msg)
	return u.(model), cmd
}

func persisted(t *testing.T, m model, path string) string {
	t.Helper()
	_, text, ok, err := m.store.LoadNote(path)
	if err != nil || !ok {
		t.Fatalf("load %s: found=%v err=%v", path, ok, err)
	}
	return text
}

func TestRollingSavePauseAndEncryption(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("# Test title\nPrivate prose for the vault"))
	if !m.saves.armed || m.saves.inFlight != nil {
		t.Fatal("edit did not debounce")
	}
	early := vaultSaveWake{epoch: m.editorEpoch, timer: m.saves.timer, at: m.saves.lastEdit.Add(100 * time.Millisecond)}
	m, _ = updateModel(m, early)
	if m.saves.inFlight != nil || !m.saves.armed {
		t.Fatal("saved before pause")
	}
	wake := vaultSaveWake{epoch: m.editorEpoch, timer: m.saves.timer, at: m.nextVaultSave()}
	m, cmd := updateModel(m, wake)
	if cmd == nil || m.saves.inFlight == nil {
		t.Fatal("pause did not start save")
	}
	m, _ = updateModel(m, cmd())
	if m.dirty || persisted(t, m, "draft.md") != "# Test title\nPrivate prose for the vault" {
		t.Fatal("save failed")
	}
	bytes, err := os.ReadFile(filepath.Join(m.cwd, "vault.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "Private prose for the vault") {
		t.Fatal("plaintext content in database")
	}
	m.undoEdit()
	if m.editorText() != "" {
		t.Fatal("save cleared undo")
	}
}

func TestContinuousTypingHasMaximumSaveInterval(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("a"))
	first := m.saves.firstDirty
	m.saves.lastEdit = first.Add(4900 * time.Millisecond)
	if !m.nextVaultSave().Equal(first.Add(vaultSaveInterval)) {
		t.Fatal("continuous typing postponed save")
	}
	m, cmd := updateModel(m, vaultSaveWake{epoch: m.editorEpoch, timer: m.saves.timer, at: first.Add(vaultSaveInterval)})
	if cmd == nil || m.saves.inFlight == nil {
		t.Fatal("maximum interval did not save")
	}
	m, _ = updateModel(m, cmd())
}

func TestSlowSaveKeepsNewerEditsDirtyAndFlushesLatest(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("first"))
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	old := m.saves.inFlight
	results := make(chan tea.Msg, 1)
	go func() { results <- cmd() }()
	for i := 0; i < 20; i++ {
		m, _ = updateModel(m, textKey("x"))
	}
	m, second := updateModel(m, <-results)
	if !m.dirty || m.saves.inFlight == nil || m.saves.inFlight == old || second == nil {
		t.Fatal("newer edits incorrectly marked saved")
	}
	want := m.editorText()
	m, _ = updateModel(m, second())
	if m.dirty || persisted(t, m, "draft.md") != want {
		t.Fatal("latest snapshot not persisted")
	}
	m, _ = updateModel(m, vaultSaveResult{snapshot: old})
	if m.dirty || m.saves.inFlight != nil {
		t.Fatal("duplicate completion changed state")
	}
}

func TestFailedSaveRetainsDraftAndCancelsTransition(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("keep this"))
	m, _ = updateModel(m, altKey('n'))
	if m.saves.pending == nil {
		t.Fatal("new note not deferred")
	}
	m, _ = updateModel(m, vaultSaveResult{snapshot: m.saves.inFlight, err: errors.New("disk full")})
	if !m.dirty || m.editorText() != "keep this" || m.saves.pending != nil || m.saves.err == "" {
		t.Fatal("failure discarded draft")
	}
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	m, _ = updateModel(m, cmd())
	if m.dirty || m.saves.err != "" || persisted(t, m, "draft.md") != "keep this" {
		t.Fatal("retry failed")
	}
}

func TestVaultTransitionsWaitForSave(t *testing.T) {
	for _, key := range []tea.KeyMsg{altKey('n'), {Type: tea.KeyCtrlQ}, {Type: tea.KeyEsc}} {
		t.Run(key.String(), func(t *testing.T) {
			m := vaultModel(t)
			m.treeVisible = true
			m, _ = updateModel(m, textKey("protect me"))
			m, cmd := updateModel(m, key)
			if m.editorText() != "protect me" || m.saves.pending == nil {
				t.Fatal("transition ran before save")
			}
			m, _ = updateModel(m, cmd())
			if persisted(t, m, "draft.md") != "protect me" {
				t.Fatal("transition lost edits")
			}
			if key.String() == "alt+n" && m.vaultPath == "draft.md" {
				t.Fatal("new note never opened")
			}
		})
	}
}

func TestAutoLockWaitsForInFlightSave(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("before lock"))
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	m.store.SetAutoLockTimeout(time.Nanosecond)
	time.Sleep(time.Millisecond)
	m, _ = updateModel(m, autoLockTickMsg{})
	if !m.store.Unlocked() || m.saves.pending == nil {
		t.Fatal("locked before save")
	}
	m, _ = updateModel(m, cmd())
	if m.store.Unlocked() || m.editorText() != "" {
		t.Fatal("did not lock after save")
	}
	if err := m.store.Unlock("disposable-password"); err != nil {
		t.Fatal(err)
	}
	if persisted(t, m, "draft.md") != "before lock" {
		t.Fatal("lock lost draft")
	}
}

func TestStaleTimerAndDiskEditsDoNotAutosave(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("old"))
	wake := vaultSaveWake{epoch: m.editorEpoch, timer: m.saves.timer, at: m.nextVaultSave()}
	m.loadEditorText("disk")
	m.isVault = false
	m, _ = updateModel(m, textKey(" edit"))
	m, cmd := updateModel(m, wake)
	if cmd != nil || m.saves.inFlight != nil || m.saves.armed {
		t.Fatal("disk or stale timer autosaved")
	}
}

func TestFailedAutoLockSaveAllowsFurtherTyping(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("before lock"))
	m.store.SetAutoLockTimeout(100 * time.Millisecond)
	time.Sleep(110 * time.Millisecond)
	m, _ = updateModel(m, autoLockTickMsg{})
	m, _ = updateModel(m, vaultSaveResult{snapshot: m.saves.inFlight, err: errors.New("disk full")})
	m, _ = updateModel(m, textKey(" more"))
	if !m.store.Unlocked() || m.editorText() != "before lock more" || m.saves.pending != nil {
		t.Fatal("failed lock save trapped the user in repeated transitions")
	}
}

func TestExpiredCleanVaultLocksBeforeAcceptingInput(t *testing.T) {
	m := vaultModel(t)
	m.loadEditorText("already saved")
	m.store.SetAutoLockTimeout(time.Nanosecond)
	time.Sleep(time.Millisecond)
	m, _ = updateModel(m, textKey("must not be inserted"))
	if m.store.Unlocked() || m.editorText() != "" || m.mode != modeVault {
		t.Fatal("input refreshed activity before enforcing expired auto-lock")
	}
}

func TestAutoLockDoesNotRecreateDetachedDraft(t *testing.T) {
	m := vaultModel(t)
	m.loadEditorText("deleted draft")
	m.vaultPath, m.filePath = "", ""
	m.dirty = true
	m.store.SetAutoLockTimeout(100 * time.Millisecond)
	time.Sleep(110 * time.Millisecond)
	m, _ = updateModel(m, autoLockTickMsg{})
	if m.saves.inFlight != nil || m.vaultPath != "" || m.saves.err == "" || !m.store.Unlocked() {
		t.Fatal("auto-lock silently saved an unattached draft")
	}
}

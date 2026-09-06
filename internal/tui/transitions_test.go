package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func diskModel(t *testing.T) model {
	t.Helper()
	m := inputModel()
	m.cwd = t.TempDir()
	m.diskPath = filepath.Join(m.cwd, "draft.md")
	m.filePath = m.diskPath
	m.treeExpanded = map[string]bool{"vault:": true, "file:": true}
	if err := os.WriteFile(m.diskPath, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	m.loadEditorText("saved")
	m, _ = updateModel(m, textKey(" new words"))
	return m
}

func TestEscRevealsTreeAndReturnsToWriting(t *testing.T) {
	m := inputModel()
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.treeVisible || m.focus != focusTree {
		t.Fatal("Esc did not reveal/focus tree")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	m, _ = updateModel(m, textKey("immediate typing"))
	if m.focus != focusEditor || m.editorText() != "immediate typing" {
		t.Fatal("return did not enable typing")
	}
}

func TestEscClosesSearchBeforeLeavingEditor(t *testing.T) {
	m := inputModel()
	m.findPrompt = textinput.New()
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeWrite || m.focus != focusEditor || m.treeVisible {
		t.Fatal("Esc from search left editor")
	}
}

func TestDiskQuitSaveDiscardCancel(t *testing.T) {
	for _, answer := range []string{"s", "d", "esc"} {
		t.Run(answer, func(t *testing.T) {
			m := diskModel(t)
			path := m.diskPath
			m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlQ})
			if m.mode != modeConfirmUnsaved || cmd != nil {
				t.Fatal("quit did not guard unsaved disk draft")
			}
			key := textKey(answer)
			if answer == "esc" {
				key = tea.KeyMsg{Type: tea.KeyEsc}
			}
			m, cmd = updateModel(m, key)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if answer == "s" && string(data) != "saved new words" {
				t.Fatal("save did not persist")
			}
			if answer != "s" && string(data) != "saved" {
				t.Fatal("discard/cancel wrote disk")
			}
			if answer == "esc" {
				if cmd != nil || !m.dirty || m.mode != modeWrite || m.editorText() != "saved new words" {
					t.Fatal("cancel did not retain draft")
				}
			} else if cmd == nil {
				t.Fatal("quit never resumed")
			}
		})
	}
}

func TestDiskSaveFailureStaysInConfirmation(t *testing.T) {
	m := diskModel(t)
	m.diskPath = m.cwd // A directory cannot be written as a document.
	m, _ = updateModel(m, altKey('n'))
	m, cmd := updateModel(m, textKey("s"))
	if m.mode != modeConfirmUnsaved || m.err == "" || !m.dirty || cmd != nil {
		t.Fatal("failed save allowed transition")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.editorText() != "saved new words" || m.mode != modeWrite {
		t.Fatal("failed save lost text")
	}
}

func TestEscLeavingDiskDoesNotSaveOrDiscard(t *testing.T) {
	m := diskModel(t)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.dirty || m.focus != focusTree || m.mode != modeWrite {
		t.Fatal("Esc should only change focus")
	}
	data, _ := os.ReadFile(m.diskPath)
	if string(data) != "saved" {
		t.Fatal("Esc autosaved disk")
	}
}

func TestDeferredVaultTransitionCanBeCancelled(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("keep writing"))
	m, cmd := updateModel(m, altKey('n'))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = updateModel(m, textKey(" after cancel"))
	m, latest := updateModel(m, cmd())
	if m.vaultPath != "draft.md" || m.saves.pending != nil || !m.dirty || latest == nil {
		t.Fatal("cancelled transition resumed or edits were lost")
	}
	m, _ = updateModel(m, latest())
	if persisted(t, m, "draft.md") != "keep writing after cancel" {
		t.Fatal("cancel interrupted saving")
	}
}

func TestVaultTreeOpenWaitsForLatestDraft(t *testing.T) {
	m := vaultModel(t)
	if err := m.store.SaveNote("other-id", "other.md", "Other", "other document"); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, textKey("save before switching"))
	m.focus = focusTree
	m.tree = []treeEntry{{id: "vault:other.md", path: "other.md", vault: true}}
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.vaultPath != "draft.md" {
		t.Fatal("switched before save")
	}
	m, _ = updateModel(m, cmd())
	// Tree refresh must retain the originally selected target.
	if m.vaultPath != "other.md" || m.editorText() != "other document" {
		t.Fatalf("opened wrong note: %s %q", m.vaultPath, m.editorText())
	}
	if persisted(t, m, "draft.md") != "save before switching" {
		t.Fatal("switch lost draft")
	}
}

func TestDeletedVaultDraftIsNotSilentlyRecreatedOnDeparture(t *testing.T) {
	m := vaultModel(t)
	m.loadEditorText("deleted document still in buffer")
	m.vaultPath, m.filePath = "", ""
	m.dirty = true
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || m.saves.inFlight != nil {
		t.Fatal("Esc recreated deleted note")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlQ})
	if m.mode != modeConfirmUnsaved {
		t.Fatal("detached draft should require an explicit save/discard choice")
	}
	m, cmd = updateModel(m, textKey("d"))
	if cmd == nil || m.saves.inFlight != nil {
		t.Fatal("discard should quit without recreating note")
	}
	notes, err := m.store.ListNotes()
	if err != nil || len(notes) != 0 {
		t.Fatal("deleted draft was saved implicitly")
	}
}

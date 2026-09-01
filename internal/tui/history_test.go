package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUndoCoalescesTypingBurst(t *testing.T) {
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: newDocumentEditor(),
	}
	m.editor.Focus()
	m.loadEditorText("")
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(model)
	updated, _ = m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(model)
	if m.editorText() != "ab" {
		t.Fatalf("typed = %q, want ab", m.editorText())
	}
	updated, _ = m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = updated.(model)
	if m.editorText() != "" {
		t.Fatalf("undo coalesced burst = %q, want empty", m.editorText())
	}
}

func TestUndoThenRedo(t *testing.T) {
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: newDocumentEditor(),
	}
	m.editor.Focus()
	m.loadEditorText("keep")
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	m = updated.(model)
	if m.editorText() != "keep!" {
		t.Fatalf("typed = %q, want keep!", m.editorText())
	}
	updated, _ = m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = updated.(model)
	if m.editorText() != "keep" {
		t.Fatalf("undo = %q, want keep", m.editorText())
	}
	m.redoEdit()
	if m.editorText() != "keep!" {
		t.Fatalf("redo = %q, want keep!", m.editorText())
	}
}

func TestLoadEditorTextClearsHistory(t *testing.T) {
	m := model{editor: newDocumentEditor(), focus: focusEditor, view: viewEdit, mode: modeWrite}
	m.editor.Focus()
	m.loadEditorText("one")
	m.recordEdit(m.editorSnapshot())
	m.setEditorText("two")
	m.loadEditorText("three")
	m.undoEdit()
	if m.editorText() != "three" {
		t.Fatalf("undo after load = %q, want three", m.editorText())
	}
}

func TestNothingToUndoStatus(t *testing.T) {
	m := model{editor: newDocumentEditor(), focus: focusEditor, view: viewEdit, mode: modeWrite}
	m.loadEditorText("x")
	m.undoEdit()
	if m.status != "nothing to undo" {
		t.Fatalf("status = %q", m.status)
	}
	if m.editorText() != "x" {
		t.Fatalf("text changed on empty undo: %q", m.editorText())
	}
}

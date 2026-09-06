package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func inputModel() model {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: newDocumentEditor()}
	m.editor.Focus()
	m.loadEditorText("")
	return m
}

func sendKey(m model, msg tea.KeyMsg) model {
	updated, _ := m.updateWrite(msg)
	return updated.(model)
}

func textKey(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestPastePreservesStructureAndHasSeparateUndo(t *testing.T) {
	for _, clipboard := range []bool{false, true} {
		m := sendKey(inputModel(), textKey("prefix"))
		text := "\t日本語\n- pasted\n"
		if clipboard {
			updated, _ := m.Update(editorPasteMsg{epoch: m.editorEpoch, text: text})
			m = updated.(model)
		} else {
			key := textKey(text)
			key.Paste = true
			m = sendKey(m, key)
		}
		if m.editorText() != "prefix"+text || !m.dirty {
			t.Fatalf("paste: %q", m.editorText())
		}
		m.undoEdit()
		if m.editorText() != "prefix" {
			t.Fatalf("paste undo: %q", m.editorText())
		}
		m.undoEdit()
		if m.editorText() != "" {
			t.Fatal("typing undo failed")
		}
	}
}

func TestLateClipboardDoesNotReachAnotherDocument(t *testing.T) {
	m := inputModel()
	epoch := m.editorEpoch
	m.loadEditorText("different note")
	updated, _ := m.Update(editorPasteMsg{epoch: epoch, text: "wrong"})
	if updated.(model).editorText() != "different note" {
		t.Fatal("stale paste inserted")
	}
}

func TestUndoSeparatesNavigationAndDeletion(t *testing.T) {
	m := sendKey(inputModel(), textKey("ab"))
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyLeft})
	m = sendKey(m, textKey("X"))
	m.undoEdit()
	if m.editorText() != "ab" {
		t.Fatalf("navigation undo: %q", m.editorText())
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnd})
	m = sendKey(m, textKey("c"))
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m.undoEdit()
	if m.editorText() != "abc" {
		t.Fatalf("deletion undo: %q", m.editorText())
	}
}

func TestOutdentPreservesOtherLinesAndCursor(t *testing.T) {
	m := inputModel()
	m.loadEditorText("\tkeep\n    word")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.editorText() != "\tkeep\nword" || m.editorCursorCol() != 4 {
		t.Fatalf("outdent: %q col=%d", m.editorText(), m.editorCursorCol())
	}
	m.undoEdit()
	if m.editorText() != "\tkeep\n    word" {
		t.Fatal("outdent undo failed")
	}
}

func TestEditorKeysAndDeferredPreview(t *testing.T) {
	m := inputModel()
	m.markdown.enabled = true
	m.preview.Width, m.preview.Height = 40, 3
	m.preview.SetContent("unchanged")
	preview := m.preview.View()
	m = sendKey(m, textKey("word next"))
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyHome})
	m = sendKey(m, altKey('d'))
	if m.mode != modeWrite || strings.Contains(m.editorText(), "word") {
		t.Fatal("alt+d did not delete word")
	}
	if m.preview.View() != preview {
		t.Fatal("preview changed")
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlY})
	if strings.Contains(m.editorText(), "word") {
		t.Fatal("ctrl+y did not redo")
	}
	if m.markdown.term != nil {
		t.Fatal("hidden preview rendered")
	}
}

func BenchmarkLongDocumentTyping(b *testing.B) {
	m := inputModel()
	m.width, m.height = 120, 40
	m.resize()
	m.loadEditorText(strings.Repeat("A paragraph with **Markdown** and several words.\n\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m = sendKey(m, textKey("x"))
		_ = m.View()
		m = sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
}

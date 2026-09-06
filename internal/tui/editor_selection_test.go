package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectReplaceAndEscape(t *testing.T) {
	m := inputModel()
	m.loadEditorText("one two")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlShiftLeft})
	if m.textBetween(m.dragStart, m.dragEnd) != "two" {
		t.Fatalf("word selection: %q", m.textBetween(m.dragStart, m.dragEnd))
	}
	m = sendKey(m, textKey("three"))
	if m.editorText() != "one three" || m.hasTextSelection() {
		t.Fatal("typing did not replace selection")
	}
	m.undoEdit()
	if m.editorText() != "one two" {
		t.Fatal("replacement undo failed")
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.hasTextSelection() || m.focus != focusEditor {
		t.Fatal("Esc left editor instead of clearing selection")
	}
}

func TestSelectionAcrossLinesIndentAndPaste(t *testing.T) {
	m := inputModel()
	m.loadEditorText("- one\n- two")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.editorText() != "    - one\n    - two" || !m.hasTextSelection() {
		t.Fatal("selection indentation failed")
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.editorText() != "- one\n- two" {
		t.Fatal("selection outdent failed")
	}
	paste := textKey("replacement\n\ttext")
	paste.Paste = true
	m = sendKey(m, paste)
	if m.editorText() != "replacement\n\ttext" {
		t.Fatalf("paste replacement: %q", m.editorText())
	}
}

func TestSelectionGraphemesAndWideText(t *testing.T) {
	m := inputModel()
	m.loadEditorText("日本e\u0301👩‍💻")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyShiftLeft})
	if m.textBetween(m.dragStart, m.dragEnd) != "👩‍💻" {
		t.Fatalf("split emoji: %q", m.textBetween(m.dragStart, m.dragEnd))
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyShiftLeft})
	if m.textBetween(m.dragStart, m.dragEnd) != "e\u0301👩‍💻" {
		t.Fatal("split combining sequence")
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.editorText() != "日本" {
		t.Fatal("Unicode selection deletion failed")
	}
}

func TestCopyCutFailureAndEyesOnly(t *testing.T) {
	m := inputModel()
	m.loadEditorText("keep text")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	a, b := m.selectionBounds()
	msg := editorCopyMsg{epoch: m.editorEpoch, source: m.editorText(), a: a, b: b, cut: true, err: errors.New("clipboard offline")}
	m, _ = updateModel(m, msg)
	if m.editorText() != "keep text" {
		t.Fatal("failed copy deleted text")
	}
	msg.err = nil
	m, _ = updateModel(m, msg)
	if m.editorText() != "" {
		t.Fatal("successful cut did not remove selection")
	}
	m.undoEdit()
	if m.editorText() != "keep text" {
		t.Fatal("cut undo failed")
	}
	m.eyesOnly = true
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	updated, cmd := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlX})
	m = updated.(model)
	if cmd != nil || m.editorText() != "keep text" {
		t.Fatal("eyes-only cut permitted")
	}
	m = sendKey(m, textKey("internal replacement"))
	if m.editorText() != "internal replacement" {
		t.Fatal("eyes-only internal editing blocked")
	}
}

func TestMouseSelectionPersistsAfterRelease(t *testing.T) {
	m := inputModel()
	m.width, m.height = 100, 30
	m.resize()
	m.loadEditorText("hello world")
	cx, cy, _, _ := m.mainContentBounds()
	u, _ := m.updateMouse(tea.MouseMsg{Action: tea.MouseActionPress, X: cx + editorGutterWidth, Y: cy})
	m = u.(model)
	u, cmd := m.updateMouse(tea.MouseMsg{Action: tea.MouseActionRelease, X: cx + editorGutterWidth + 5, Y: cy})
	m = u.(model)
	if cmd != nil || !m.hasTextSelection() {
		t.Fatal("drag did not retain editable selection")
	}
	m = sendKey(m, textKey("goodbye"))
	if m.editorText() != "goodbye world" {
		t.Fatalf("mouse replacement: %q", m.editorText())
	}
}

func TestCopyDoesNotQuitOrFlushVault(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("draft"))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.saves.pending != nil || m.saves.inFlight != nil {
		t.Fatal("Ctrl+C acted as quit")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.saves.pending != nil || m.focus != focusEditor {
		t.Fatal("selection Esc triggered departure")
	}
}

func TestMultilineSelectionPasteCaret(t *testing.T) {
	for _, newline := range []string{"\n", "\r", "\r\n"} {
		t.Run(newline, func(t *testing.T) {
			m := inputModel()
			m.loadEditorText("old text")
			m = sendKey(m, tea.KeyMsg{Type: tea.KeyCtrlA})
			paste := textKey("> > ```" + newline + "> > - literal")
			paste.Paste = true
			m = sendKey(m, paste)
			m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			m = sendKey(m, textKey("code"))
			if got := m.editorText(); got != "> > ```\n> > - literal\n> > code" {
				t.Fatalf("paste caret: %q", got)
			}
		})
	}
}

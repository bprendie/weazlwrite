package tui

import (
	"strings"
	"testing"
)

func TestFindNextFromCursorColumnOnSameLine(t *testing.T) {
	m := model{editor: newDocumentEditor(), view: viewEdit, focus: focusEditor}
	m.editor.Focus()
	m.setEditorText("foo bar foo")
	m.editor.SetCursor(0)
	m.findInEditor("foo")
	if m.editor.Line() != 0 {
		t.Fatalf("line = %d, want 0", m.editor.Line())
	}
	if m.editorCursorCol() != 8 {
		t.Fatalf("col = %d, want 8 (second foo)", m.editorCursorCol())
	}
	if strings.Contains(m.status, "wrapped") {
		t.Fatalf("status = %q, did not expect wrap", m.status)
	}
}

func TestFindWrapsAroundAndReportsIt(t *testing.T) {
	m := model{editor: newDocumentEditor(), view: viewEdit, focus: focusEditor}
	m.editor.Focus()
	m.setEditorText("zzz\nfoo")
	m.moveEditorToLine(1)
	m.editor.SetCursor(len([]rune("foo")))
	m.findInEditor("zzz")
	if m.editor.Line() != 0 {
		t.Fatalf("line = %d, want 0", m.editor.Line())
	}
	if m.editorCursorCol() != 0 {
		t.Fatalf("col = %d, want 0", m.editorCursorCol())
	}
	if !strings.Contains(m.status, "wrapped") {
		t.Fatalf("status = %q, want wrapped", m.status)
	}
}

func TestFindNextInLinesHelpers(t *testing.T) {
	lines := []string{"alpha foo", "beta", "foo gamma"}
	line, col, wrapped, ok := findNextInLines(lines, "foo", 0, 1)
	if !ok || line != 0 || col != 6 || wrapped {
		t.Fatalf("same-line next = %d/%d wrapped=%v ok=%v", line, col, wrapped, ok)
	}
	line, col, wrapped, ok = findNextInLines(lines, "foo", 0, 7)
	if !ok || line != 2 || col != 0 || wrapped {
		t.Fatalf("later-line = %d/%d wrapped=%v ok=%v", line, col, wrapped, ok)
	}
	line, col, wrapped, ok = findNextInLines(lines, "alpha", 2, 1)
	if !ok || line != 0 || col != 0 || !wrapped {
		t.Fatalf("wrap = %d/%d wrapped=%v ok=%v", line, col, wrapped, ok)
	}
}

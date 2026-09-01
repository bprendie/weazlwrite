package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWrapRunesMatchesTextareaLineHeight(t *testing.T) {
	cases := []string{
		"",
		"short",
		strings.Repeat("word ", 40),
		strings.Repeat("x", 80),
		"# heading with a fairly long title that should wrap",
		"https://example.com/this/is/a/very/long/url/without/spaces/at/all",
	}
	for _, width := range []int{10, 20, 40} {
		ed := newDocumentEditor()
		ed.SetWidth(width + editorGutterWidth)
		ed.Focus()
		for _, value := range cases {
			ed.SetValue(value)
			got := len(wrapRunes([]rune(value), ed.Width()))
			want := ed.LineInfo().Height
			if got != want {
				t.Fatalf("width %d value %q: wrap rows %d, textarea Height %d", width, value, got, want)
			}
		}
	}
}

func TestWrappedParagraphSpansVisualPages(t *testing.T) {
	m := model{
		styles: newStyles(),
		view:   viewEdit,
		focus:  focusEditor,
		editor: newDocumentEditor(),
	}
	m.editor.Focus()
	m.editor.SetWidth(24)
	m.editor.SetHeight(4)
	m.editor.SetValue(strings.Repeat("word ", 80))
	m.editor.SetCursor(0)
	if m.totalPages() < 2 {
		t.Fatalf("pages = %d, want a wrapped paragraph to span pages", m.totalPages())
	}
	start := m.editorVisualRow()
	m.editorPageDown()
	if m.editorVisualRow() <= start {
		t.Fatal("page down did not advance a visual row")
	}
	m.jumpToPage(2)
	if m.currentPage() != 2 {
		t.Fatalf("current page = %d, want 2", m.currentPage())
	}
}

func TestHomeStaysLogicalOnWrappedLine(t *testing.T) {
	ed := newDocumentEditor()
	ed.Focus()
	ed.SetWidth(20)
	ed.SetValue(strings.Repeat("word ", 20))
	ed.CursorDown()
	ed.CursorDown()
	if ed.LineInfo().RowOffset == 0 {
		t.Fatal("expected cursor on a wrapped continuation row")
	}
	ed, _ = ed.Update(tea.KeyMsg{Type: tea.KeyHome})
	ed, _ = ed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if !strings.HasPrefix(ed.Value(), "X") {
		t.Fatalf("home then X value = %q, want insert at paragraph start", ed.Value())
	}
}

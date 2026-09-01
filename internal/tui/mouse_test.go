package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTextBetweenSameLineAndMultiLine(t *testing.T) {
	m := model{editor: newDocumentEditor()}
	m.setEditorText("hello world\nsecond")
	got := m.textBetween(textPos{line: 0, col: 6}, textPos{line: 0, col: 11})
	if got != "world" {
		t.Fatalf("same line = %q, want world", got)
	}
	got = m.textBetween(textPos{line: 0, col: 6}, textPos{line: 1, col: 3})
	if got != "world\nsec" {
		t.Fatalf("multi line = %q, want world\\nsec", got)
	}
}

func TestClickWithoutDragDoesNotCopy(t *testing.T) {
	m := model{
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusEditor,
		view:         viewEdit,
		treeVisible:  false,
		mouseCapture: true,
		width:        80,
		height:       24,
		editor:       newDocumentEditor(),
		editorChrome: &editorChrome{gutter: editorGutterWidth},
	}
	m.editor.Focus()
	m.editor.SetPromptFunc(editorGutterWidth, m.editorChrome.prompt)
	m.setEditorText("hello world")
	m.resize()
	cx, cy, _, _ := m.mainContentBounds()
	press := tea.MouseMsg{Action: tea.MouseActionPress, X: cx + editorGutterWidth + 1, Y: cy}
	updated, cmd := m.updateMouse(press)
	m = updated.(model)
	if !m.editorDrag {
		t.Fatal("press did not start editor drag")
	}
	release := tea.MouseMsg{Action: tea.MouseActionRelease, X: cx + editorGutterWidth + 1, Y: cy}
	updated, cmd = m.updateMouse(release)
	m = updated.(model)
	if m.editorDrag {
		t.Fatal("release left editorDrag on")
	}
	if cmd != nil {
		t.Fatal("click without drag issued a copy command")
	}
}

func TestTreeClickSelectsRow(t *testing.T) {
	m := model{
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusEditor,
		view:         viewEdit,
		treeVisible:  true,
		mouseCapture: true,
		width:        80,
		height:       24,
		editor:       newDocumentEditor(),
		tree: []treeEntry{
			{name: "Vault", id: "vault:", isDir: true, vault: true},
			{name: "a.md", id: "vault:a.md", path: "a.md", vault: true, depth: 1},
			{name: "b.md", id: "vault:b.md", path: "b.md", vault: true, depth: 1},
		},
		treeExpanded: map[string]bool{"vault:": true, "file:": true},
	}
	m.resize()
	cx, cy, _, _ := m.treeContentBounds()
	press := tea.MouseMsg{Action: tea.MouseActionPress, X: cx, Y: cy + 2}
	updated, _ := m.updateMouse(press)
	m = updated.(model)
	if m.focus != focusTree {
		t.Fatalf("focus = %v, want tree", m.focus)
	}
	if m.treeIdx != 2 {
		t.Fatalf("treeIdx = %d, want 2", m.treeIdx)
	}
}

func TestToggleMouseCaptureDisablesMouse(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, mouseCapture: true, editor: newDocumentEditor()}
	updated, cmd := m.toggleMouseCapture()
	got := updated.(model)
	if got.mouseCapture {
		t.Fatal("expected mouse capture off")
	}
	if cmd == nil {
		t.Fatal("expected disable-mouse command")
	}
}

func TestEyesOnlyKeepsMouseCaptureOn(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, mouseCapture: true, eyesOnly: true, editor: newDocumentEditor()}
	updated, _ := m.toggleMouseCapture()
	got := updated.(model)
	if !got.mouseCapture {
		t.Fatal("eyes only allowed mouse capture off")
	}
}

func TestRuneIndexAtVisualOnWrappedLine(t *testing.T) {
	line := strings.Repeat("word ", 10)
	idx := runeIndexAtVisual(line, 0, 0, 20)
	if idx != 0 {
		t.Fatalf("row0 col0 idx = %d, want 0", idx)
	}
	idx = runeIndexAtVisual(line, 1, 0, 20)
	if idx <= 0 {
		t.Fatalf("row1 col0 idx = %d, want > 0", idx)
	}
}

func TestTextareaYOffsetReadable(t *testing.T) {
	ed := newDocumentEditor()
	ed.Focus()
	ed.SetWidth(20)
	ed.SetHeight(3)
	ed.SetValue(strings.Repeat("line\n", 20) + "end")
	for ed.Line() > 0 {
		ed.CursorUp()
	}
	_ = ed.View()
	for i := 0; i < 15; i++ {
		ed, _ = ed.Update(tea.KeyMsg{Type: tea.KeyDown})
		_ = ed.View()
	}
	if textareaYOffset(ed) == 0 {
		t.Fatal("expected viewport to scroll after moving down a long buffer")
	}
}

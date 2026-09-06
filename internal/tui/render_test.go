package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestViewDoesNotExceedTerminalWidth(t *testing.T) {
	for _, size := range []struct {
		width  int
		height int
	}{
		{width: 80, height: 24},
		{width: 100, height: 30},
		{width: 120, height: 40},
	} {
		editor := textarea.New()
		editor.SetValue("# Test\n\nBody")
		m := model{
			styles:  newStyles(),
			mode:    modeWrite,
			focus:   focusEditor,
			width:   size.width,
			height:  size.height,
			editor:  editor,
			preview: viewport.New(0, 0),
			status:  "editing test.md",
			treeExpanded: map[string]bool{
				"vault:": true,
				"file:":  true,
			},
			tree: []treeEntry{
				{name: "Vault", id: "vault:", isDir: true, vault: true},
				{name: "Files", id: "file:", isDir: true},
				{name: "test.md", id: "file:test.md", path: "test.md", depth: 1},
			},
		}
		m.resize()
		m.renderPreview()

		for i, line := range strings.Split(m.View(), "\n") {
			if got := lipgloss.Width(line); got > size.width {
				t.Fatalf("%dx%d line %d width = %d, want <= %d: %q", size.width, size.height, i+1, got, size.width, line)
			}
		}
		if got := lipgloss.Height(m.View()); got > size.height {
			t.Fatalf("%dx%d height = %d, want <= %d", size.width, size.height, got, size.height)
		}
	}
}

func TestFullLogoRendersAtEightyColumns(t *testing.T) {
	got := renderLogo(ansiHeader(), 80)
	if !strings.Contains(got, "_______________.__") {
		t.Fatalf("expected full ASCII logo at 80 columns, got %q", got)
	}
}

func TestTabIndentsEditorAndEntersFromTree(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(model)
	if got.focus != focusEditor || got.editorText() != "    " {
		t.Fatalf("tab should indent editor: focus=%v text=%q", got.focus, got.editorText())
	}
	got.setFocus(focusTree)
	updated, _ = got.updateWrite(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(model)
	if got.focus != focusEditor {
		t.Fatalf("tab from tree focus = %v, want editor", got.focus)
	}
}

func TestEscapeMovesFromRenderToEdit(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusPreview, view: viewRender, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.view != viewEdit {
		t.Fatalf("esc from render view = %v, want edit", got.view)
	}
	if got.focus != focusEditor {
		t.Fatalf("esc from render focus = %v, want editor", got.focus)
	}
}

func TestEscapeMovesFromEditorToTree(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.focus != focusTree {
		t.Fatalf("esc from editor focus = %v, want tree", got.focus)
	}
}

func TestAltIOpensAIPrompt(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, aiPrompt: textinput.New(), editor: textarea.New()}
	updated, _ := m.updateWrite(altKey('i'))
	got := updated.(model)
	if got.mode != modeAI {
		t.Fatalf("alt+i mode = %v, want modeAI", got.mode)
	}
}

func TestPlainHTypesInEditorInsteadOfOpeningHelp(t *testing.T) {
	editor := textarea.New()
	editor.Focus()
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: editor}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(model)
	if got.mode != modeWrite {
		t.Fatalf("plain h mode = %v, want modeWrite", got.mode)
	}
	if got.editor.Value() != "h" {
		t.Fatalf("editor value = %q, want h", got.editor.Value())
	}
}

func TestPlainHStillOpensHelpOutsideEditorTyping(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusTree, view: viewEdit, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(model)
	if got.mode != modeHelp {
		t.Fatalf("plain h from tree mode = %v, want modeHelp", got.mode)
	}
}

func TestCtrlRSwitchesToRenderMode(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlR})
	got := updated.(model)
	if got.view != viewRender || got.focus != focusPreview {
		t.Fatalf("ctrl+r view/focus = %v/%v, want render/preview", got.view, got.focus)
	}
}

func TestLayoutWidthsForTreeStates(t *testing.T) {
	m := model{width: 100, height: 24, styles: newStyles(), treeVisible: true, focus: focusEditor}
	treeW, mainW := m.layoutWidths()
	if treeW != 25 || mainW != 75 {
		t.Fatalf("compact layout = %d/%d, want 25/75", treeW, mainW)
	}
	m.focus = focusTree
	treeW, mainW = m.layoutWidths()
	if treeW != 55 || mainW != 45 {
		t.Fatalf("focused layout = %d/%d, want 55/45", treeW, mainW)
	}
	m.treeVisible = false
	treeW, mainW = m.layoutWidths()
	if treeW != 0 || mainW != 100 {
		t.Fatalf("hidden tree layout = %d/%d, want 0/100", treeW, mainW)
	}
}

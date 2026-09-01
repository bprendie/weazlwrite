package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestEnterWorksPastNinetyNineLines(t *testing.T) {
	ed := newDocumentEditor()
	ed.Focus()
	ed.SetValue(strings.Repeat("line\n", 119) + "line")
	if ed.LineCount() != 120 {
		t.Fatalf("line count = %d, want 120", ed.LineCount())
	}
	ed, _ = ed.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if ed.LineCount() != 121 {
		t.Fatalf("after enter line count = %d, want 121", ed.LineCount())
	}
}

func TestSetEditorTextPreservesTabs(t *testing.T) {
	m := model{editor: newDocumentEditor()}
	m.setEditorText("func\tmain")
	if got := m.editorText(); got != "func\tmain" {
		t.Fatalf("editorText = %q, want func\\tmain", got)
	}
	if strings.Contains(m.editor.Value(), "    ") {
		t.Fatal("textarea expanded tabs to spaces")
	}
}

func TestEditorWrapsToPaneNotDefaultMaxWidth(t *testing.T) {
	m := model{
		styles:       newStyles(),
		width:        600,
		height:       24,
		treeVisible:  false,
		focus:        focusEditor,
		view:         viewEdit,
		editor:       newDocumentEditor(),
		editorChrome: &editorChrome{gutter: editorGutterWidth},
	}
	m.editor.SetPromptFunc(editorGutterWidth, m.editorChrome.prompt)
	m.resize()
	if m.editor.Width() <= 500-editorGutterWidth {
		t.Fatalf("wrap width = %d, still looks capped at 500", m.editor.Width())
	}
	m.setEditorText(strings.Repeat("word ", 400))
	m.prepareEditorView()
	paneW := contentWidth(m.styles.panel, 600)
	widest := 0
	rows := 0
	for _, line := range strings.Split(m.editor.View(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows++
		widest = max(widest, lipgloss.Width(line))
	}
	if rows < 2 {
		t.Fatalf("long line produced %d visible rows, want wrapped", rows)
	}
	if widest > paneW {
		t.Fatalf("editor view width %d > pane %d", widest, paneW)
	}
}

func TestGutterHasStableWidth(t *testing.T) {
	got := formatGutter(1, editorGutterWidth)
	if len(got) != editorGutterWidth {
		t.Fatalf("gutter 1 = %q width %d, want %d", got, len(got), editorGutterWidth)
	}
	got = formatGutter(9999, editorGutterWidth)
	if len(got) != editorGutterWidth {
		t.Fatalf("gutter 9999 = %q width %d, want %d", got, len(got), editorGutterWidth)
	}
	if w := lipgloss.Width(styleGutter(got)); w != editorGutterWidth {
		t.Fatalf("styled gutter width = %d, want %d", w, editorGutterWidth)
	}
}

func TestEditorViewDoesNotBlankBetweenLines(t *testing.T) {
	ed := newDocumentEditor()
	ed.Focus()
	ed.SetPromptFunc(editorGutterWidth, func(i int) string {
		return formatGutter(i+1, editorGutterWidth)
	})
	ed.SetWidth(40)
	ed.SetHeight(6)
	ed.SetValue("alpha\nbeta\ngamma")
	lines := strings.Split(ed.View(), "\n")
	if len(lines) < 3 {
		t.Fatalf("view lines = %d", len(lines))
	}
	got := []string{
		strings.TrimSpace(ansi.Strip(lines[0])),
		strings.TrimSpace(ansi.Strip(lines[1])),
		strings.TrimSpace(ansi.Strip(lines[2])),
	}
	want := []string{"1alpha", "2beta", "3gamma"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEditorPromptUsesTreeFolderPurple(t *testing.T) {
	ed := newDocumentEditor()
	s := newStyles()
	if ed.FocusedStyle.Prompt.GetForeground() != s.treeFolder.GetForeground() {
		t.Fatalf("editor prompt color = %v, want tree folder %v", ed.FocusedStyle.Prompt.GetForeground(), s.treeFolder.GetForeground())
	}
}

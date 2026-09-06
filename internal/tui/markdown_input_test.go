package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestMarkdownEnter(t *testing.T) {
	for _, tc := range []struct{ before, after string }{
		{"- one", "- one\n- "}, {"+ one", "+ one\n+ "}, {"9. nine", "9. nine\n10. "},
		{"12) twelve", "12) twelve\n13) "}, {"  - [X] done", "  - [X] done\n  - [ ] "},
		{"> words", "> words\n> "}, {"> - one", "> - one\n> - "},
		{"- ", ""}, {"    - ", "- "}, {"> - ", "> "}, {"> ", ""}, {"> > ", "> "},
		{"```\n  - literal", "```\n  - literal\n  "}, {"~~~go\n1. literal", "~~~go\n1. literal\n"},
		{"> > ```\n> > - literal", "> > ```\n> > - literal\n> > "},
		{"```\ncode\n```\n- prose", "```\ncode\n```\n- prose\n- "},
	} {
		t.Run(tc.before, func(t *testing.T) {
			m := inputModel()
			m.loadEditorText(tc.before)
			m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			if m.editorText() != tc.after {
				t.Fatalf("got %q want %q", m.editorText(), tc.after)
			}
			m.undoEdit()
			if m.editorText() != tc.before {
				t.Fatal("assisted enter was not one undo step")
			}
		})
	}
}

func TestMarkdownSplitNestAndBackspace(t *testing.T) {
	m := inputModel()
	m.loadEditorText("- first second")
	m.editor.SetCursor(8)
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.editorText() != "- first \n- second" {
		t.Fatalf("split: %q", m.editorText())
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.editorText() != "- first \n    - second" {
		t.Fatalf("nest: %q", m.editorText())
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.editorText() != "- first \n- second" {
		t.Fatalf("unnest: %q", m.editorText())
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.editorText() != "- first \nsecond" {
		t.Fatalf("remove marker: %q", m.editorText())
	}
}

func TestMarkdownPasteDoesNotContinueLists(t *testing.T) {
	m := inputModel()
	m.loadEditorText("- ")
	key := textKey("one\n  3. three\n```\n\t- code")
	key.Paste = true
	m = sendKey(m, key)
	if m.editorText() != "- one\n  3. three\n```\n\t- code" {
		t.Fatal("paste transformed Markdown")
	}
}

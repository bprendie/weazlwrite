package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

const (
	editorGutterWidth = 5
	tabSentinel       = '\uE000'
)

type editorChrome struct {
	gutter  int
	prompts []string
}

func gutterColor() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(neonViolet)
}

func styleGutter(s string) string {
	return gutterColor().Render(s)
}

func (c *editorChrome) prompt(i int) string {
	if c == nil {
		return ""
	}
	if i >= 0 && i < len(c.prompts) {
		return c.prompts[i]
	}
	return strings.Repeat(" ", c.gutter)
}

func (c *editorChrome) refresh(value string, wrapWidth int) {
	if c == nil {
		return
	}
	if c.gutter <= 0 {
		c.gutter = editorGutterWidth
	}
	c.prompts = gutterPrompts(value, wrapWidth, c.gutter)
}

func gutterPrompts(value string, wrapWidth, gutter int) []string {
	lines := strings.Split(value, "\n")
	out := make([]string, 0, len(lines))
	blank := strings.Repeat(" ", gutter)
	for i, line := range lines {
		rows := wrapRunes([]rune(line), wrapWidth)
		out = append(out, formatGutter(i+1, gutter))
		for j := 1; j < len(rows); j++ {
			out = append(out, blank)
		}
	}
	return out
}

func formatGutter(n, width int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) >= width {
		return s[len(s)-width:]
	}
	return fmt.Sprintf("%*s", width, s)
}

func newDocumentEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "# Untitled\n\nStart writing..."
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.MaxWidth = 0
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.KeyMap = newEditorKeyMap()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(panelAlt)
	ta.FocusedStyle.LineNumber = gutterColor()
	ta.FocusedStyle.Prompt = gutterColor()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle().Foreground(muted)
	ta.BlurredStyle.LineNumber = gutterColor()
	ta.BlurredStyle.Prompt = gutterColor()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(ink).Background(panel)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(ink).Background(panel)
	return ta
}

func (m *model) configureEditor() {
	m.editor.MaxHeight = 0
	m.editor.MaxWidth = 0
	m.editor.ShowLineNumbers = false
	m.editor.Prompt = ""
	if m.editorChrome == nil {
		m.editorChrome = &editorChrome{gutter: editorGutterWidth}
	}
	m.editor.SetPromptFunc(m.editorChrome.gutter, m.editorChrome.prompt)
}

func (m model) prepareEditorView() {
	if m.editorChrome == nil {
		return
	}
	m.editorChrome.refresh(m.editor.Value(), max(1, m.editor.Width()))
}

func (m model) editorText() string {
	return restoreTabs(m.editor.Value())
}

func (m *model) setEditorText(s string) {
	m.editor.SetValue(hideTabs(s))
}

func hideTabs(s string) string {
	return strings.ReplaceAll(s, "\t", string(tabSentinel))
}

func restoreTabs(s string) string {
	return strings.ReplaceAll(s, string(tabSentinel), "\t")
}

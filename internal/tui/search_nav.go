package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startFind() (tea.Model, tea.Cmd) {
	m.mode = modeFind
	m.findPrompt.SetValue(m.lastFind)
	m.findPrompt.Focus()
	m.editor.Blur()
	m.status = "find"
	return m, textinput.Blink
}

func (m model) updateFind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.findPrompt.Value())
		if query == "" {
			m.err = "find text is required"
			return m, nil
		}
		m.lastFind = query
		m.mode = modeWrite
		m.setMainFocus()
		m.findNext(query)
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "find cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.findPrompt, cmd = m.findPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) startJumpPage() (tea.Model, tea.Cmd) {
	m.mode = modeJumpPage
	m.jumpPrompt.SetValue(strconv.Itoa(max(1, m.currentPage())))
	m.jumpPrompt.Focus()
	m.editor.Blur()
	m.status = "jump to page"
	return m, textinput.Blink
}

func (m model) updateJumpPage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		page, err := strconv.Atoi(strings.TrimSpace(m.jumpPrompt.Value()))
		if err != nil || page < 1 {
			m.err = "page must be a positive number"
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		m.jumpToPage(page)
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "jump cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.jumpPrompt, cmd = m.jumpPrompt.Update(msg)
		return m, cmd
	}
}

func (m *model) findNext(query string) {
	if m.view == viewRender {
		m.findInPreview(query)
		return
	}
	m.findInEditor(query)
}

func (m *model) findInPreview(query string) {
	rendered := m.markdown.Render(m.editor.Value(), max(10, m.preview.Width))
	lines := strings.Split(rendered, "\n")
	q := strings.ToLower(query)
	start := min(len(lines), m.preview.YOffset+1)
	for pass := 0; pass < 2; pass++ {
		from := 0
		if pass == 0 {
			from = start
		}
		for i := from; i < len(lines); i++ {
			if strings.Contains(strings.ToLower(lines[i]), q) {
				m.preview.SetYOffset(max(0, i-1))
				m.err = ""
				m.status = fmt.Sprintf("found %q on page %d", query, m.currentPage())
				return
			}
		}
	}
	m.err = "not found: " + query
}

func (m *model) findInEditor(query string) {
	lines := strings.Split(m.editor.Value(), "\n")
	q := strings.ToLower(query)
	start := min(len(lines), m.editor.Line()+1)
	for pass := 0; pass < 2; pass++ {
		from := 0
		if pass == 0 {
			from = start
		}
		for i := from; i < len(lines); i++ {
			col := strings.Index(strings.ToLower(lines[i]), q)
			if col >= 0 {
				m.moveEditorToLine(i)
				m.editor.SetCursor(col)
				m.err = ""
				m.status = fmt.Sprintf("found %q on page %d", query, m.currentPage())
				return
			}
		}
	}
	m.err = "not found: " + query
}

func (m *model) jumpToPage(page int) {
	page = min(max(1, page), max(1, m.totalPages()))
	if m.view == viewRender {
		m.preview.SetYOffset((page - 1) * max(1, m.preview.Height))
	} else {
		m.moveEditorToLine((page - 1) * max(1, m.editor.Height()))
	}
	m.err = ""
	m.status = fmt.Sprintf("page %d of %d", m.currentPage(), m.totalPages())
}

func (m model) currentPage() int {
	if m.view == viewRender {
		return min(max(1, m.preview.YOffset/max(1, m.preview.Height)+1), max(1, m.totalPages()))
	}
	return min(max(1, m.editor.Line()/max(1, m.editor.Height())+1), max(1, m.totalPages()))
}

func (m model) totalPages() int {
	if m.view == viewRender {
		return max(1, (m.preview.TotalLineCount()+max(1, m.preview.Height)-1)/max(1, m.preview.Height))
	}
	return max(1, (m.editor.LineCount()+max(1, m.editor.Height())-1)/max(1, m.editor.Height()))
}

func (m *model) moveEditorToLine(line int) {
	line = min(max(0, line), max(0, m.editor.LineCount()-1))
	for m.editor.Line() < line {
		m.editor.CursorDown()
	}
	for m.editor.Line() > line {
		m.editor.CursorUp()
	}
	m.editor.SetCursor(0)
}

func (m *model) editorPageUp() {
	target := max(0, m.editor.Line()-max(1, m.editor.Height()))
	m.moveEditorToLine(target)
}

func (m *model) editorPageDown() {
	target := min(max(0, m.editor.LineCount()-1), m.editor.Line()+max(1, m.editor.Height()))
	m.moveEditorToLine(target)
}

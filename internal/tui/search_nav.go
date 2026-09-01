package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

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
	rendered := m.markdown.Render(m.editorText(), max(10, m.preview.Width))
	lines := strings.Split(rendered, "\n")
	line, _, wrapped, ok := findNextInLines(lines, query, m.preview.YOffset, 0)
	if !ok {
		m.err = "not found: " + query
		return
	}
	m.preview.SetYOffset(max(0, line))
	m.err = ""
	m.status = findStatus(query, m.currentPage(), wrapped)
}

func (m *model) findInEditor(query string) {
	lines := strings.Split(m.editorText(), "\n")
	line, col, wrapped, ok := findNextInLines(lines, query, m.editor.Line(), m.editorCursorCol()+1)
	if !ok {
		m.err = "not found: " + query
		return
	}
	m.moveEditorToLine(line)
	m.editor.SetCursor(col)
	m.err = ""
	m.status = findStatus(query, m.currentPage(), wrapped)
}

func (m model) editorCursorCol() int {
	info := m.editor.LineInfo()
	return max(0, info.StartColumn+info.ColumnOffset)
}

func findStatus(query string, page int, wrapped bool) string {
	msg := fmt.Sprintf("found %q on page %d", query, page)
	if wrapped {
		msg += " (wrapped)"
	}
	return msg
}

func findNextInLines(lines []string, query string, startLine, startCol int) (line, col int, wrapped, ok bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" || len(lines) == 0 {
		return 0, 0, false, false
	}
	startLine = min(max(0, startLine), len(lines)-1)
	startCol = max(0, startCol)
	if c, found := indexFrom(lines[startLine], q, startCol); found {
		return startLine, c, false, true
	}
	for i := startLine + 1; i < len(lines); i++ {
		if c, found := indexFrom(lines[i], q, 0); found {
			return i, c, false, true
		}
	}
	for i := 0; i < startLine; i++ {
		if c, found := indexFrom(lines[i], q, 0); found {
			return i, c, true, true
		}
	}
	if c, found := indexFrom(lines[startLine], q, 0); found && c < startCol {
		return startLine, c, true, true
	}
	return 0, 0, false, false
}

func indexFrom(line, query string, fromRune int) (int, bool) {
	rs := []rune(strings.ToLower(line))
	if fromRune < 0 {
		fromRune = 0
	}
	if fromRune > len(rs) || query == "" {
		return 0, false
	}
	hay := string(rs[fromRune:])
	at := strings.Index(hay, query)
	if at < 0 {
		return 0, false
	}
	return fromRune + utf8.RuneCountInString(hay[:at]), true
}

func (m *model) jumpToPage(page int) {
	page = min(max(1, page), max(1, m.totalPages()))
	if m.view == viewRender {
		m.preview.SetYOffset((page - 1) * max(1, m.preview.Height))
	} else {
		m.moveEditorToVisualRow((page - 1) * max(1, m.editor.Height()))
	}
	m.err = ""
	m.status = fmt.Sprintf("page %d of %d", m.currentPage(), m.totalPages())
}

func (m model) currentPage() int {
	if m.view == viewRender {
		return min(max(1, m.preview.YOffset/max(1, m.preview.Height)+1), max(1, m.totalPages()))
	}
	return min(max(1, m.editorVisualRow()/max(1, m.editor.Height())+1), max(1, m.totalPages()))
}

func (m model) totalPages() int {
	if m.view == viewRender {
		return max(1, (m.preview.TotalLineCount()+max(1, m.preview.Height)-1)/max(1, m.preview.Height))
	}
	h := max(1, m.editor.Height())
	return max(1, (m.editorVisualRowCount()+h-1)/h)
}

func (m model) editorWrapWidth() int {
	return max(1, m.editor.Width())
}

func (m model) editorVisualRowCount() int {
	return visualRowCount(m.editor.Value(), m.editorWrapWidth())
}

func (m model) editorVisualRow() int {
	return cursorVisualRow(m.editor.Value(), m.editor.Line(), m.editor.LineInfo().RowOffset, m.editorWrapWidth())
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

func (m *model) moveEditorToVisualRow(target int) {
	total := m.editorVisualRowCount()
	target = min(max(0, target), max(0, total-1))
	line, offset := logicalAtVisualRow(m.editor.Value(), target, m.editorWrapWidth())
	m.moveEditorToLine(line)
	for i := 0; i < offset; i++ {
		m.editor.CursorDown()
	}
}

func (m *model) editorPageUp() {
	m.moveEditorToVisualRow(m.editorVisualRow() - max(1, m.editor.Height()))
}

func (m *model) editorPageDown() {
	m.moveEditorToVisualRow(m.editorVisualRow() + max(1, m.editor.Height()))
}

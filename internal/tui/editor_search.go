package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type searchMatch struct{ start, end textPos }
type editorSearchState struct {
	origin       textPos
	originOffset int
	matches      []searchMatch
	index        int
	replace      bool
	replacement  textinput.Model
}

func (m model) startEditorSearch() (tea.Model, tea.Cmd) {
	m.search = editorSearchState{origin: m.cursorTextPos(), originOffset: m.editor.YOffset(), index: -1, replacement: textinput.New()}
	m.search.replacement.Placeholder = "replacement (empty deletes)"
	m.clearTextSelection()
	m.undoOpen = false
	m.mode = modeFind
	m.findPrompt.SetValue(m.lastFind)
	m.findPrompt.Focus()
	m.editor.Blur()
	m.resize()
	m.refreshEditorMatches(m.search.origin)
	return m, textinput.Blink
}

func (m model) updateEditorSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeWrite
		m.lastFind = m.findPrompt.Value()
		m.setMainFocus()
		m.placeEditorCursor(m.search.origin)
		m.editor.SetYOffset(m.search.originOffset)
		m.status = "find closed"
		m.err = ""
		return m, nil
	case "f2":
		m.mode = modeWrite
		m.lastFind = m.findPrompt.Value()
		m.setMainFocus()
		m.selectCurrentMatch()
		return m, nil
	case "tab", "shift+tab":
		m.search.replace = !m.search.replace
		if m.search.replace {
			m.findPrompt.Blur()
			m.search.replacement.Focus()
		} else {
			m.search.replacement.Blur()
			m.findPrompt.Focus()
		}
		return m, textinput.Blink
	case "enter":
		if m.search.replace {
			m.replaceCurrentMatch()
		} else {
			m.stepEditorMatch(1)
		}
		return m, nil
	case "shift+enter", "shift+f3", "f15", "alt+f3":
		m.stepEditorMatch(-1)
		return m, nil
	case "f3":
		m.stepEditorMatch(1)
		return m, nil
	default:
		var cmd tea.Cmd
		if m.search.replace {
			m.search.replacement, cmd = m.search.replacement.Update(msg)
		} else {
			before := m.findPrompt.Value()
			m.findPrompt, cmd = m.findPrompt.Update(msg)
			if before != m.findPrompt.Value() {
				m.refreshEditorMatches(m.search.origin)
			}
		}
		return m, cmd
	}
}

func (m *model) refreshEditorMatches(from textPos) {
	query := m.findPrompt.Value()
	m.lastFind = query
	m.search.matches = nil
	m.search.index = -1
	m.err = ""
	if query == "" {
		return
	}
	length := len([]rune(query))
	for row, line := range strings.Split(m.editorText(), "\n") {
		for col := 0; ; {
			at, ok := indexFrom(line, strings.ToLower(query), col)
			if !ok {
				break
			}
			m.search.matches = append(m.search.matches, searchMatch{textPos{row, at}, textPos{row, at + length}})
			col = at + length
		}
	}
	if len(m.search.matches) == 0 {
		m.status = "no matches"
		return
	}
	m.search.index = 0
	for i, match := range m.search.matches {
		if !match.start.less(from) {
			m.search.index = i
			break
		}
	}
	m.showCurrentMatch()
}

func (m *model) showCurrentMatch() {
	if m.search.index < 0 || m.search.index >= len(m.search.matches) {
		return
	}
	m.placeEditorCursor(m.search.matches[m.search.index].start)
	m.status = fmt.Sprintf("match %d/%d", m.search.index+1, len(m.search.matches))
}

func (m *model) stepEditorMatch(direction int) {
	n := len(m.search.matches)
	if n == 0 {
		return
	}
	m.search.index = (m.search.index + direction + n) % n
	m.showCurrentMatch()
}

func (m *model) selectCurrentMatch() {
	if m.search.index < 0 || m.search.index >= len(m.search.matches) {
		return
	}
	match := m.search.matches[m.search.index]
	m.dragStart, m.dragEnd = match.start, match.end
	m.placeEditorCursor(match.end)
}

func (m *model) replaceCurrentMatch() {
	if m.search.index < 0 || m.search.index >= len(m.search.matches) {
		return
	}
	match := m.search.matches[m.search.index]
	before := m.editorSnapshot()
	m.undoOpen = false
	m.replaceTextRange(match.start, match.end, m.search.replacement.Value())
	m.recordEdit(before)
	m.undoOpen = false
	m.dirty = true
	m.refreshEditorMatches(m.cursorTextPos())
}

func (m *model) repeatEditorFind(backward bool) {
	if m.lastFind == "" {
		m.status = "Ctrl+F starts a search"
		return
	}
	from := m.cursorTextPos()
	if m.hasTextSelection() {
		from, _ = m.selectionBounds()
	}
	m.findPrompt.SetValue(m.lastFind)
	m.clearTextSelection()
	m.refreshEditorMatches(from)
	if backward {
		m.stepEditorMatch(-1)
	} else if m.search.index >= 0 && m.search.matches[m.search.index].start.eq(from) {
		m.stepEditorMatch(1)
	}
	m.selectCurrentMatch()
}

func (m model) editorSearchView(width int) string {
	find, replace := m.findPrompt, m.search.replacement
	find.Width = max(5, width-26)
	replace.Width = max(5, width-26)
	count := fmt.Sprintf(" [%d/%d]", max(0, m.search.index+1), len(m.search.matches))
	first := "Find: " + find.View() + count
	second := "Tab replace | Enter/F3 next | Shift+F3 previous | F2 edit match | Esc close"
	if m.search.replace {
		second = "Replace: " + replace.View() + "  Enter replaces; Tab finds"
	}
	return ansi.Truncate(first, width, "") + "\n" + ansi.Truncate(second, width, "")
}

package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) hasTextSelection() bool { return !m.dragStart.eq(m.dragEnd) }
func (m *model) clearTextSelection() {
	m.dragStart, m.dragEnd = textPos{}, textPos{}
	m.editorDrag = false
}
func (m model) cursorTextPos() textPos { return textPos{m.editor.Line(), m.editorCursorCol()} }

func (m model) selectionBounds() (textPos, textPos) {
	a, b := m.dragStart, m.dragEnd
	if b.less(a) {
		a, b = b, a
	}
	return a, b
}

func (m *model) replaceTextRange(a, b textPos, text string) {
	lines := strings.Split(m.editorText(), "\n")
	start, end := posToOffset(lines, a), posToOffset(lines, b)
	runes := []rune(m.editorText())
	m.setEditorText(string(runes[:start]) + string(runes[end:]))
	m.editor.SetPosition(textarea.Position{Line: a.line, Column: a.col})
	m.editor.InsertString(hideTabs(text))
	m.clearTextSelection()
}

func (m *model) replaceSelection(text string) {
	a, b := m.selectionBounds()
	m.replaceTextRange(a, b, text)
}

func (m *model) applyEditorHighlights() {
	var ranges []textarea.Highlight
	if m.mode == modeFind && m.view == viewEdit {
		for i, match := range m.search.matches {
			style := lipgloss.NewStyle().Background(neonViolet).Foreground(ink)
			if i == m.search.index {
				style = lipgloss.NewStyle().Background(neonCyan).Foreground(panel)
			}
			ranges = append(ranges, textarea.Highlight{Start: textarea.Position{Line: match.start.line, Column: match.start.col}, End: textarea.Position{Line: match.end.line, Column: match.end.col}, Style: style})
		}
	}
	if m.hasTextSelection() {
		a, b := m.selectionBounds()
		ranges = append(ranges, textarea.Highlight{Start: textarea.Position{Line: a.line, Column: a.col}, End: textarea.Position{Line: b.line, Column: b.col}, Style: lipgloss.NewStyle().Reverse(true)})
	}
	m.editor.SetHighlights(ranges)
}

func (m *model) extendEditorSelection(key string) bool {
	keys := map[string]tea.KeyType{
		"shift+left": tea.KeyLeft, "shift+right": tea.KeyRight,
		"shift+up": tea.KeyUp, "shift+down": tea.KeyDown,
		"shift+home": tea.KeyHome, "shift+end": tea.KeyEnd,
		"ctrl+shift+left": tea.KeyCtrlLeft, "ctrl+shift+right": tea.KeyCtrlRight,
		"ctrl+shift+home": tea.KeyCtrlHome, "ctrl+shift+end": tea.KeyCtrlEnd,
	}
	typeKey, ok := keys[key]
	if !ok {
		return false
	}
	m.undoOpen = false
	if !m.hasTextSelection() {
		m.dragStart = m.cursorTextPos()
	}
	if typeKey == tea.KeyLeft || typeKey == tea.KeyRight {
		direction := 1
		if typeKey == tea.KeyLeft {
			direction = -1
		}
		m.editor.SetPosition(m.editor.AdjacentPosition(direction))
	} else {
		m.editor, _ = m.editor.Update(tea.KeyMsg{Type: typeKey})
	}
	m.dragEnd = m.cursorTextPos()
	return true
}

func (m *model) indentSelection(outdent bool) {
	a, b := m.selectionBounds()
	last := b.line
	if b.col == 0 && last > a.line {
		last--
	}
	lines := strings.Split(m.editorText(), "\n")
	for i := a.line; i <= last; i++ {
		delta := editorIndentWidth
		if outdent {
			n := len(leadingIndent(lines[i]))
			n = min(n, editorIndentWidth)
			if strings.HasPrefix(lines[i], "\t") {
				n = 1
			}
			lines[i] = lines[i][n:]
			delta = -n
		} else {
			lines[i] = strings.Repeat(" ", editorIndentWidth) + lines[i]
		}
		if i == a.line {
			a.col = max(0, a.col+delta)
		}
		if i == b.line {
			b.col = max(0, b.col+delta)
		}
	}
	reverse := m.dragEnd.less(m.dragStart)
	m.setEditorText(strings.Join(lines, "\n"))
	m.dragStart, m.dragEnd = a, b
	if reverse {
		m.dragStart, m.dragEnd = b, a
	}
	m.editor.SetPosition(textarea.Position{Line: m.dragEnd.line, Column: m.dragEnd.col})
}

func (m model) editorDragPos(x, y int) (textPos, bool) {
	cx, cy, cw, ch := m.mainContentBounds()
	if y < cy {
		m.editor.ScrollRows(-1)
		y = cy
	} else if y >= cy+ch {
		m.editor.ScrollRows(1)
		y = cy + ch - 1
	}
	x = min(max(x, cx), cx+cw-1)
	return m.editorPosAt(x, y)
}

type editorCopyMsg struct {
	epoch  uint64
	source string
	a, b   textPos
	cut    bool
	err    error
}

func (m model) copyEditorSelection(cut bool) (tea.Model, tea.Cmd) {
	if m.eyesOnly {
		m.err = "eyes only notes cannot be copied or cut"
		return m, nil
	}
	if !m.hasTextSelection() {
		m.status = "select text to copy"
		return m, nil
	}
	a, b := m.selectionBounds()
	result := editorCopyMsg{epoch: m.editorEpoch, source: m.editorText(), a: a, b: b, cut: cut}
	text := m.textBetween(a, b)
	return m, func() tea.Msg { result.err = clipboard.WriteAll(text); return result }
}

func (m model) finishEditorCopy(msg editorCopyMsg) (tea.Model, tea.Cmd) {
	if msg.epoch != m.editorEpoch {
		return m, nil
	}
	if msg.err != nil {
		m.err = "clipboard: " + msg.err.Error()
		return m, nil
	}
	m.err = ""
	m.status = "copied selection"
	a, b := m.selectionBounds()
	if msg.cut && !m.eyesOnly && msg.source == m.editorText() && a.eq(msg.a) && b.eq(msg.b) {
		before := m.editorSnapshot()
		m.undoOpen = false
		m.replaceSelection("")
		m.recordEdit(before)
		m.undoOpen = false
		m.dirty = true
		m.status = "cut selection"
	}
	return m, nil
}

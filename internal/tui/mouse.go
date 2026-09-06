package tui

import (
	"strings"

	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	mouse := tea.MouseEvent(msg)
	if mouse.IsWheel() {
		return m.handleWheel(mouse)
	}
	switch mouse.Action {
	case tea.MouseActionPress:
		return m.handleMousePress(mouse)
	case tea.MouseActionMotion:
		return m.handleMouseMotion(mouse)
	case tea.MouseActionRelease:
		return m.handleMouseRelease(mouse)
	}
	return m, nil
}

func (m model) handleWheel(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	target := m.focusAtXY(mouse.X, mouse.Y)
	if target == focusTree {
		switch mouse.Type {
		case tea.MouseWheelUp:
			m.scrollTree(-3)
		case tea.MouseWheelDown:
			m.scrollTree(3)
		}
		return m, nil
	}
	if target == focusEditor && m.view == viewEdit {
		delta := 3
		if mouse.Type == tea.MouseWheelUp {
			delta = -3
		}
		m.editor.ScrollRows(delta)
		return m, nil
	}
	if target == focusPreview {
		switch mouse.Type {
		case tea.MouseWheelUp:
			m.preview.ScrollUp(3)
		case tea.MouseWheelDown:
			m.preview.ScrollDown(3)
		}
		return m, nil
	}
	return m, nil
}

func (m model) handleMousePress(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	target := m.focusAtXY(mouse.X, mouse.Y)
	if target == focusEditor && m.view == viewEdit {
		pos, ok := m.editorPosAt(mouse.X, mouse.Y)
		m.setFocus(target)
		if ok {
			m.placeEditorCursor(pos)
			m.editorDrag = true
			m.dragStart = pos
			m.dragEnd = pos
		}
		return m, nil
	}
	m.setFocus(target)
	m.editorDrag = false
	if target == focusTree {
		m.clickTreeAt(mouse.X, mouse.Y)
	}
	return m, nil
}

func (m model) handleMouseMotion(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	if !m.editorDrag || m.view != viewEdit {
		return m, nil
	}
	if pos, ok := m.editorDragPos(mouse.X, mouse.Y); ok {
		m.dragEnd = pos
		m.placeEditorCursor(pos)
	}
	return m, nil
}

func (m model) handleMouseRelease(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	if !m.editorDrag {
		return m, nil
	}
	if pos, ok := m.editorPosAt(mouse.X, mouse.Y); ok {
		m.dragEnd = pos
		m.placeEditorCursor(pos)
	}
	m.editorDrag = false
	if m.dragStart.eq(m.dragEnd) {
		return m, nil
	}
	m.status = "text selected; type to replace, Ctrl+C copies"
	return m, nil
}

func (m model) focusAtXY(x, y int) focus {
	return m.focusAtX(x)
}

func (m *model) clickTreeAt(x, y int) {
	cx, cy, cw, ch := m.treeContentBounds()
	if x < cx || x >= cx+cw || y < cy || y >= cy+ch {
		return
	}
	idx := m.treeOffset + (y - cy)
	if idx < 0 || idx >= len(m.tree) {
		return
	}
	m.treeIdx = idx
	m.ensureTreeSelectionVisible()
}

func (m model) treeContentBounds() (x, y, width, height int) {
	headerLines := len(strings.Split(renderLogo(ansiHeader(), max(20, m.width)), "\n"))
	bodyY := headerLines + 1
	treeW, _ := m.layoutWidths()
	return 2, bodyY + 1, contentWidth(m.styles.sidebar, treeW), m.treeContentHeight()
}

func (m model) editorPosAt(x, y int) (textPos, bool) {
	cx, cy, cw, ch := m.mainContentBounds()
	if x < cx || x >= cx+cw || y < cy || y >= cy+ch {
		return textPos{}, false
	}
	relY := y - cy
	colX := (x - cx) - editorGutterWidth
	if colX < 0 {
		colX = 0
	}
	vis := textareaYOffset(m.editor) + relY
	return m.visualToPos(vis, colX), true
}

func (m model) visualToPos(vis, colX int) textPos {
	p := m.editor.PositionAt(vis, colX)
	return textPos{line: p.Line, col: p.Column}
}

func (m *model) placeEditorCursor(pos textPos) {
	m.undoOpen = false
	m.editor.SetPosition(textarea.Position{Line: pos.line, Column: pos.col})
	if m.editor.Focused() {
		m.editor, _ = m.editor.Update(nil)
	}
}

func (m model) textBetween(a, b textPos) string {
	if b.less(a) {
		a, b = b, a
	}
	lines := strings.Split(m.editorText(), "\n")
	if len(lines) == 0 {
		return ""
	}
	a.line = min(max(0, a.line), len(lines)-1)
	b.line = min(max(0, b.line), len(lines)-1)
	aRunes := []rune(lines[a.line])
	bRunes := []rune(lines[b.line])
	a.col = min(max(0, a.col), len(aRunes))
	b.col = min(max(0, b.col), len(bRunes))
	if a.line == b.line {
		if a.col >= b.col {
			return ""
		}
		return string(aRunes[a.col:b.col])
	}
	var bld strings.Builder
	bld.WriteString(string(aRunes[a.col:]))
	bld.WriteByte('\n')
	for i := a.line + 1; i < b.line; i++ {
		bld.WriteString(lines[i])
		bld.WriteByte('\n')
	}
	bld.WriteString(string(bRunes[:b.col]))
	return bld.String()
}

func textareaYOffset(ed textarea.Model) int { return ed.YOffset() }

func posToOffset(lines []string, pos textPos) int {
	n := 0
	for i := 0; i < pos.line && i < len(lines); i++ {
		n += len([]rune(lines[i])) + 1
	}
	if pos.line >= 0 && pos.line < len(lines) {
		n += min(max(0, pos.col), len([]rune(lines[pos.line])))
	}
	return n
}

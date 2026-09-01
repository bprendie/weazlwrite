package tui

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(tea.MouseMsg(mouse))
		return m, cmd
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
	if pos, ok := m.editorPosAt(mouse.X, mouse.Y); ok {
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
	if m.eyesOnly {
		m.status = "eyes only notes cannot be copied"
		m.err = "eyes only notes keep copy protection on"
		return m, nil
	}
	text := m.textBetween(m.dragStart, m.dragEnd)
	if strings.TrimSpace(text) == "" {
		m.status = "selection empty"
		return m, nil
	}
	m.status = "copied selection"
	return m, copyOSC52(text)
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
	value := m.editor.Value()
	width := m.editorWrapWidth()
	line, rowOff := logicalAtVisualRow(value, vis, width)
	lines := strings.Split(value, "\n")
	if line < 0 || line >= len(lines) {
		return textPos{}
	}
	return textPos{line: line, col: runeIndexAtVisual(lines[line], rowOff, colX, width)}
}

func (m *model) placeEditorCursor(pos textPos) {
	m.moveEditorToLine(pos.line)
	m.editor.SetCursor(pos.col)
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

func textareaYOffset(ed textarea.Model) int {
	vp := reflect.ValueOf(ed).FieldByName("viewport")
	if !vp.IsValid() || vp.Kind() != reflect.Pointer || vp.IsNil() {
		return 0
	}
	y := vp.Elem().FieldByName("YOffset")
	if !y.IsValid() || !y.CanInt() {
		return 0
	}
	return int(y.Int())
}

func (m model) editorDragView(width, height int) string {
	value := m.editor.Value()
	lines := strings.Split(value, "\n")
	wrapW := m.editorWrapWidth()
	offset := textareaYOffset(m.editor)
	start, end := m.dragStart, m.dragEnd
	if end.less(start) {
		start, end = end, start
	}
	a := posToOffset(lines, start)
	b := posToOffset(lines, end)
	var (
		out    []string
		vis    int
		docOff int
	)
	for li, line := range lines {
		runes := []rune(line)
		rows := wrapRunes(runes, wrapW)
		idx := 0
		for ri, row := range rows {
			trimmed := []rune(strings.TrimRight(string(row), " "))
			if vis >= offset && len(out) < height {
				gutter := formatGutter(li+1, editorGutterWidth)
				if ri > 0 {
					gutter = strings.Repeat(" ", editorGutterWidth)
				}
				out = append(out, gutter+paintRunes(trimmed, docOff+idx, a, b))
			}
			idx += len(trimmed)
			if idx < len(runes) && unicode.IsSpace(runes[idx]) {
				idx++
			}
			vis++
		}
		docOff += len(runes) + 1
	}
	return strings.Join(out, "\n")
}

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

func paintRunes(runes []rune, docOff, a, b int) string {
	if len(runes) == 0 {
		return ""
	}
	var bld strings.Builder
	for i, r := range runes {
		ch := string(r)
		off := docOff + i
		if off >= a && off < b {
			ch = lipgloss.NewStyle().Reverse(true).Render(ch)
		}
		bld.WriteString(ch)
	}
	return bld.String()
}

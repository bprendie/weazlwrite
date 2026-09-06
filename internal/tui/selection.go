package tui

import (
	"encoding/base64"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m model) updateSelectionMouse(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	if mouse.IsWheel() {
		return m, nil
	}
	row, ok := m.selectionRowAt(mouse.X, mouse.Y)
	if !ok && !m.selecting {
		if mouse.Action == tea.MouseActionPress {
			m.status = "selection mode: drag inside the writing pane"
		}
		return m, nil
	}
	if !ok && m.selecting {
		row = m.scrollSelectionForMouse(mouse.Y)
	}
	_, _, width, _ := m.mainContentBounds()
	logical := m.selectionLogicalAt(m.selectOffset+row, width)
	switch mouse.Action {
	case tea.MouseActionPress:
		m.selecting = true
		m.selectStart = selectPoint{row: logical}
		m.selectEnd = m.selectStart
		m.status = "selecting"
	case tea.MouseActionMotion:
		if m.selecting {
			m.selectEnd = selectPoint{row: logical}
		}
	case tea.MouseActionRelease:
		if !m.selecting {
			return m, nil
		}
		m.selectEnd = selectPoint{row: logical}
		m.selecting = false
		text := m.selectedText()
		if strings.TrimSpace(text) == "" {
			m.status = "selection empty"
			return m, nil
		}
		m.status = "copied selection"
		return m, copyOSC52(text)
	}
	return m, nil
}

func (m *model) initSelectionOffset() {
	_, _, width, height := m.mainContentBounds()
	vis := m.selectionVisualRows(width)
	cur := 0
	if m.view == viewRender {
		cur = m.preview.YOffset
	} else {
		cur = m.selectionCursorVisualRow(width)
	}
	m.selectOffset = min(max(0, cur-height/2), max(0, len(vis)-height))
}

func (m *model) scrollSelectionForMouse(y int) int {
	_, contentY, width, contentH := m.mainContentBounds()
	vis := m.selectionVisualRows(width)
	maxOffset := max(0, len(vis)-contentH)
	switch {
	case y < contentY:
		m.selectOffset = max(0, m.selectOffset-1)
		return 0
	case y >= contentY+contentH:
		m.selectOffset = min(maxOffset, m.selectOffset+1)
		return max(0, contentH-1)
	default:
		return min(max(0, y-contentY), max(0, contentH-1))
	}
}

func (m model) selectionRowAt(x, y int) (int, bool) {
	contentX, contentY, contentW, contentH := m.mainContentBounds()
	if x < contentX || x >= contentX+contentW || y < contentY || y >= contentY+contentH {
		return 0, false
	}
	return y - contentY, true
}

func (m model) mainContentBounds() (x, y, width, height int) {
	headerLines := len(strings.Split(renderLogo(ansiHeader(), max(20, m.width)), "\n"))
	bodyY := headerLines + 1
	treeW, mainW := m.layoutWidths()
	mainX := 0
	if m.treeVisible {
		mainX = treeW
	}
	contentX := mainX + 2
	contentY := bodyY + 1
	if m.view == viewEdit {
		w, pad := m.editorLayout()
		return contentX + pad, contentY, w, m.editor.Height()
	}
	return contentX, contentY, contentWidth(m.styles.activePanel, mainW), contentHeight(m.styles.activePanel, m.bodyHeight())
}

func (m model) selectedText() string {
	lines := m.selectionSourceLines()
	if len(lines) == 0 {
		return ""
	}
	start, end := m.selectionRows()
	start = min(max(0, start), len(lines)-1)
	end = min(max(0, end), len(lines)-1)
	if end < start {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n")
}

func (m model) selectionRows() (start, end int) {
	start = m.selectStart.row
	end = m.selectEnd.row
	if end < start {
		start, end = end, start
	}
	return start, end
}

func (m model) selectionSourceLines() []string {
	if m.view == viewRender {
		return m.renderedPlainLines()
	}
	return strings.Split(m.editorText(), "\n")
}

func (m model) renderedPlainLines() []string {
	rendered := m.markdown.Render(m.editorText(), m.preview.Width)
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimRight(ansi.Strip(line), " "))
	}
	return out
}

type selectionVisRow struct {
	logical int
	text    string
}

func (m model) selectionView(width, height int) string {
	vis := m.selectionVisualRows(width)
	if len(vis) == 0 {
		return ""
	}
	start, end := m.selectionRows()
	visible := make([]string, 0, height)
	for absolute := m.selectOffset; absolute < len(vis) && len(visible) < height; absolute++ {
		line := vis[absolute].text
		if vis[absolute].logical >= start && vis[absolute].logical <= end {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		visible = append(visible, line)
	}
	return strings.Join(visible, "\n")
}

func (m model) selectionVisualRows(width int) []selectionVisRow {
	lines := m.selectionSourceLines()
	out := make([]selectionVisRow, 0, len(lines))
	for i, line := range lines {
		for _, row := range wrapDisplayLine(line, width) {
			out = append(out, selectionVisRow{logical: i, text: row})
		}
	}
	return out
}

func (m model) selectionLogicalAt(visualRow, width int) int {
	vis := m.selectionVisualRows(width)
	if len(vis) == 0 {
		return 0
	}
	visualRow = min(max(0, visualRow), len(vis)-1)
	return vis[visualRow].logical
}

func (m model) selectionCursorVisualRow(width int) int {
	lines := m.selectionSourceLines()
	line := min(max(0, m.editor.Line()), max(0, len(lines)-1))
	row := 0
	for i := 0; i < line; i++ {
		row += len(wrapDisplayLine(lines[i], width))
	}
	return row
}

func wrapDisplayLine(line string, width int) []string {
	line = strings.ReplaceAll(line, "\t", "    ")
	if width < 1 {
		width = 1
	}
	rows := wrapRunes([]rune(line), width)
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, strings.TrimRight(string(row), " "))
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func copyOSC52(text string) tea.Cmd {
	return func() tea.Msg {
		tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			return nil
		}
		defer tty.Close()
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		_, _ = tty.WriteString("\x1b]52;c;" + encoded + "\x07")
		return nil
	}
}

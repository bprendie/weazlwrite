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
	absoluteRow := m.selectOffset + row
	switch mouse.Action {
	case tea.MouseActionPress:
		m.selecting = true
		m.selectStart = selectPoint{row: absoluteRow}
		m.selectEnd = m.selectStart
		m.status = "selecting"
	case tea.MouseActionMotion:
		if m.selecting {
			m.selectEnd = selectPoint{row: absoluteRow}
		}
	case tea.MouseActionRelease:
		if !m.selecting {
			return m, nil
		}
		m.selectEnd = selectPoint{row: absoluteRow}
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
	_, _, _, height := m.mainContentBounds()
	if m.view == viewRender {
		m.selectOffset = min(max(0, m.preview.YOffset), max(0, len(m.selectionSourceLines())-height))
		return
	}
	m.selectOffset = min(max(0, m.editor.Line()-height/2), max(0, len(m.selectionSourceLines())-height))
}

func (m *model) scrollSelectionForMouse(y int) int {
	_, contentY, _, contentH := m.mainContentBounds()
	lines := m.selectionSourceLines()
	maxOffset := max(0, len(lines)-contentH)
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
	return strings.Split(m.editor.Value(), "\n")
}

func (m model) renderedPlainLines() []string {
	rendered := m.markdown.Render(m.editor.Value(), m.preview.Width)
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimRight(ansi.Strip(line), " "))
	}
	return out
}

func (m model) selectionView(width, height int) string {
	lines := m.selectionSourceLines()
	if len(lines) == 0 {
		return ""
	}
	start, end := m.selectionRows()
	visible := make([]string, 0, height)
	for absolute := m.selectOffset; absolute < len(lines) && len(visible) < height; absolute++ {
		line := strings.ReplaceAll(lines[absolute], "\t", "    ")
		line = ansi.Truncate(line, max(1, width), "")
		line = strings.TrimRight(line, " ")
		if absolute >= start && absolute <= end {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		visible = append(visible, line)
	}
	return strings.Join(visible, "\n")
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

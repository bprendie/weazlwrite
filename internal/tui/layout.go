package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) bodyHeight() int {
	headerLines := len(strings.Split(renderLogo(ansiHeader(), max(20, m.width)), "\n"))
	return max(1, m.height-headerLines-2)
}

func (m model) layoutWidths() (treeW, mainW int) {
	innerW := max(20, m.width)
	if !m.treeVisible {
		return 0, innerW
	}
	if m.focus == focusTree {
		treeW = focusedTreeWidth(innerW)
	} else {
		treeW = compactTreeWidth(innerW)
	}
	mainW = max(1, innerW-treeW)
	return treeW, mainW
}

func compactTreeWidth(innerW int) int {
	return min(30, max(18, innerW/4))
}

func focusedTreeWidth(innerW int) int {
	if innerW <= 34 {
		return min(innerW-1, max(18, innerW-8))
	}
	return min(max(34, innerW*55/100), min(72, innerW-12))
}

func renderPanel(style lipgloss.Style, outerW, outerH int, content string) string {
	return style.
		Width(contentWidth(style, outerW)).
		Height(contentHeight(style, outerH)).
		MaxWidth(outerW).
		MaxHeight(outerH).
		Render(content)
}

func contentWidth(style lipgloss.Style, outerW int) int {
	return max(1, outerW-style.GetHorizontalFrameSize())
}

func contentHeight(style lipgloss.Style, outerH int) int {
	return max(1, outerH-style.GetVerticalFrameSize())
}

package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	screenW := max(20, m.width)
	screenH := max(8, m.height)
	header := renderLogo(ansiHeader(), screenW)
	status := m.status
	if m.dirty {
		status += " *"
	}
	if m.err != "" {
		status = m.styles.error.Render("! " + m.err)
	} else {
		status = m.styles.status.Render(status)
	}

	var body string
	if m.mode == modeVault {
		body = renderPanel(m.styles.panel, screenW, m.bodyHeight(), "Encrypted markdown vault\n\n"+m.password.View())
	} else if m.mode == modeAI {
		body = m.aiPromptView()
	} else if m.mode == modeGenerating {
		body = m.generatingView()
	} else {
		body = m.writeView()
	}
	help := m.styles.help.Width(screenW).Render(m.helpText())
	return m.styles.frame.Width(screenW).Height(screenH).Render(strings.Join([]string{header, status, body, help}, "\n"))
}

func (m model) writeView() string {
	innerW := max(20, m.width)
	innerH := m.bodyHeight()
	treeW := min(30, max(18, innerW/4))
	workW := max(20, innerW-treeW)
	editorW := max(10, workW/2)
	previewW := max(10, innerW-treeW-editorW)

	treeStyle := m.styles.sidebar
	editorStyle := m.styles.panel
	previewStyle := m.styles.panel
	if m.focus == focusTree {
		treeStyle = m.styles.sidebar.BorderForeground(neonCyan)
	}
	if m.focus == focusEditor {
		editorStyle = m.styles.activePanel
	}
	if m.focus == focusPreview {
		previewStyle = m.styles.activePanel
	}

	tree := renderPanel(treeStyle, treeW, innerH, m.treeView(contentWidth(treeStyle, treeW), contentHeight(treeStyle, innerH)))
	editor := renderPanel(editorStyle, editorW, innerH, m.editor.View())
	preview := renderPanel(previewStyle, previewW, innerH, m.preview.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, editor, preview)
}

func (m model) aiPromptView() string {
	w := max(20, m.width)
	popupWidth := min(76, max(30, w-4))
	copy := "AI insert prompt\n\n" + m.aiPrompt.View() + "\n\n" + m.styles.help.Render("The generated Markdown block will be inserted at the editor cursor.")
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) generatingView() string {
	w := max(20, m.width)
	popupWidth := min(60, max(30, w-4))
	copy := "AI is generating a Markdown block..."
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonViolet).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) treeView(width, height int) string {
	if len(m.tree) == 0 {
		return m.styles.sidebarDim.Render("empty")
	}
	var b strings.Builder
	for i, entry := range m.tree {
		if i >= height {
			break
		}
		name := entry.name
		if !entry.vault && entry.isDir && entry.path == m.cwd {
			name = filepath.Base(entry.path) + "/"
			if name == "./" || name == "/" {
				name = entry.path
			}
		}
		name = minString(name, max(1, width))
		line := name
		if i == m.treeIdx {
			line = m.styles.sidebarSel.Render("> " + minString(name, max(1, width-2)))
		} else if entry.isDir {
			line = m.styles.sidebarDim.Render("  " + minString(name, max(1, width-2)))
		} else {
			line = "  " + minString(name, max(1, width-2))
		}
		b.WriteString(line)
		if i != len(m.tree)-1 && i != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) helpText() string {
	if m.mode == modeVault {
		return "enter unlock/create | ctrl+c quit"
	}
	if m.mode == modeAI {
		return "enter generate | esc cancel | ctrl+c quit"
	}
	if m.mode == modeGenerating {
		return "waiting for local model | ctrl+c quit"
	}
	target := "vault"
	if !m.isVault && m.filePath != "" {
		target = fmt.Sprintf("disk:%s", m.filePath)
	}
	return "ctrl+s save " + target + " | ctrl+i ai insert | ctrl+n new vault note | ctrl+o files | ctrl+e editor | ctrl+p preview | pgup/pgdn scroll preview | ctrl+c quit"
}

func (m model) bodyHeight() int {
	headerLines := len(strings.Split(renderLogo(ansiHeader(), max(20, m.width)), "\n"))
	return max(3, m.height-headerLines-3)
}

func renderPanel(style lipgloss.Style, outerW, outerH int, content string) string {
	return style.
		Width(contentWidth(style, outerW)).
		Height(contentHeight(style, outerH)).
		Render(content)
}

func contentWidth(style lipgloss.Style, outerW int) int {
	return max(1, outerW-style.GetHorizontalFrameSize())
}

func contentHeight(style lipgloss.Style, outerH int) int {
	return max(1, outerH-style.GetVerticalFrameSize())
}

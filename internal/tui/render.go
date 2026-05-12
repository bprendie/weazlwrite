package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	screenW := max(20, m.width)
	screenH := max(8, m.height)
	header := renderLogo(ansiHeader(), screenW)
	statusText := m.status
	if m.dirty {
		statusText += " *"
	}
	statusText = strings.ReplaceAll(statusText, "\n", " ")
	statusText = minString(statusText, max(1, screenW))
	status := ""
	if m.err != "" {
		status = m.styles.error.Inline(true).MaxWidth(screenW).Render("! " + strings.ReplaceAll(m.err, "\n", " "))
	} else {
		status = m.styles.status.Inline(true).MaxWidth(screenW).Render(statusText)
	}

	var body string
	if m.mode == modeVault {
		body = renderPanel(m.styles.panel, screenW, m.bodyHeight(), "Encrypted markdown vault\n\n"+m.password.View())
	} else if m.mode == modeAI {
		body = m.aiPromptView()
	} else if m.mode == modeGenerating {
		body = m.generatingView()
	} else if m.mode == modeSaveFile {
		body = m.saveFileView()
	} else if m.mode == modeSaveVault {
		body = m.saveVaultView()
	} else {
		body = m.writeView()
	}
	body = lipgloss.NewStyle().MaxWidth(screenW).MaxHeight(m.bodyHeight()).Render(body)
	help := m.styles.help.Inline(true).MaxWidth(screenW).Render(m.helpText())
	out := strings.Join([]string{header, status, body, help}, "\n")
	return m.styles.frame.Width(screenW).Height(screenH).MaxWidth(screenW).MaxHeight(screenH).Render(out)
}

func (m model) writeView() string {
	innerH := m.bodyHeight()
	treeW, mainW := m.layoutWidths()

	treeStyle := m.styles.sidebar
	if m.focus == focusTree {
		treeStyle = m.styles.sidebar.BorderForeground(neonCyan)
	}

	mainContent := m.editor.View()
	if m.view == viewRender {
		mainContent = m.preview.View()
	}
	main := renderPanel(m.styles.activePanel, mainW, innerH, mainContent)
	if !m.treeVisible {
		return main
	}
	tree := renderPanel(treeStyle, treeW, innerH, m.treeView(contentWidth(treeStyle, treeW), contentHeight(treeStyle, innerH)))
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, main)
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
	copy := fmt.Sprintf("%s %s", m.working.View(), m.thinkingPhrase())
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonViolet).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) thinkingPhrase() string {
	if len(modelThinkingPhrases) == 0 || m.generatingAt.IsZero() {
		return "model_is_thinking"
	}
	phase := min(2, int(time.Since(m.generatingAt)/(20*time.Second)))
	start := int((m.generatingAt.UnixNano() / int64(time.Millisecond)) % int64(len(modelThinkingPhrases)))
	idx := (start + phase) % len(modelThinkingPhrases)
	return modelThinkingPhrases[idx]
}

func (m model) saveFileView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	copy := "Save to:\n\n" + m.filePrompt.View()
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) saveVaultView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	copy := "Save to:\n\n" + m.vaultPrompt.View()
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
	start := min(max(0, m.treeOffset), max(0, len(m.tree)-1))
	for row, i := 0, start; i < len(m.tree); row, i = row+1, i+1 {
		if row >= height {
			break
		}
		entry := m.tree[i]
		name := m.treeEntryLabel(entry)
		name = minString(name, max(1, width-2))
		line := "  " + name
		if i == m.treeIdx {
			line = m.styles.sidebarSel.Render("> " + name)
		} else if entry.isDir {
			line = m.styles.sidebarDim.Render(line)
		}
		b.WriteString(line)
		if i != len(m.tree)-1 && row != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) treeEntryLabel(entry treeEntry) string {
	indent := strings.Repeat("  ", entry.depth)
	current := entry.id == m.currentTreeID()
	marker := " "
	if current {
		marker = "•"
		if m.dirty {
			marker = "*"
		}
	}
	if entry.isDir {
		icon := "▸"
		if m.treeExpanded[entry.id] {
			icon = "▾"
		}
		return indent + icon + " " + entry.name
	}
	return indent + marker + " " + entry.name
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
	if m.mode == modeSaveFile {
		return "enter save | esc cancel | ctrl+c quit"
	}
	if m.mode == modeSaveVault {
		return "enter save encrypted | esc cancel | ctrl+c quit"
	}
	target := "vault"
	if !m.isVault && m.filePath != "" {
		target = fmt.Sprintf("disk:%s", m.filePath)
	}
	mode := "edit"
	if m.view == viewRender {
		mode = "render"
	}
	tree := "tree:on"
	if !m.treeVisible {
		tree = "tree:off"
	}
	return mode + " " + tree + " | space fold | ^E edit | ^R render | ^O tree | ^S " + target + " | ^V vault | ^F file | ^P AI | ^C"
}

func (m model) bodyHeight() int {
	headerLines := len(strings.Split(renderLogo(ansiHeader(), max(20, m.width)), "\n"))
	return max(1, m.height-headerLines-2)
}

func (m model) layoutWidths() (treeW, mainW int) {
	innerW := max(20, m.width)
	if !m.treeVisible {
		return 0, innerW
	}
	treeW = min(30, max(18, innerW/4))
	mainW = max(1, innerW-treeW)
	return treeW, mainW
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

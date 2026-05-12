package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	header := renderLogo(ansiHeader(), max(20, m.width-6))
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
		body = m.styles.panel.Width(max(20, m.width-6)).Render("Encrypted markdown vault\n\n" + m.password.View())
	} else {
		body = m.writeView()
	}
	help := m.styles.help.Render(m.helpText())
	return m.styles.frame.Width(m.width).Height(m.height).Render(strings.Join([]string{header, status, body, help}, "\n"))
}

func (m model) writeView() string {
	innerW := max(20, m.width-6)
	innerH := max(8, m.height-10)
	treeW := min(30, max(20, innerW/4))
	workW := max(20, innerW-treeW-2)
	editorW := max(20, workW/2)
	previewW := max(20, workW-editorW-1)

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

	tree := treeStyle.Width(treeW).Height(innerH).Render(m.treeView(treeW-2, innerH-2))
	editor := editorStyle.Width(editorW).Height(innerH).Render(m.editor.View())
	preview := previewStyle.Width(previewW).Height(innerH).Render(m.preview.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, editor, preview)
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
	target := "vault"
	if !m.isVault && m.filePath != "" {
		target = fmt.Sprintf("disk:%s", m.filePath)
	}
	return "ctrl+s save " + target + " | ctrl+n new vault note | ctrl+o files | ctrl+e editor | ctrl+p preview | pgup/pgdn scroll preview | ctrl+c quit"
}

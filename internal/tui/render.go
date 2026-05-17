package tui

import (
	"fmt"
	"strings"

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
	if m.mode == modeWrite {
		statusText += fmt.Sprintf(" | page %d/%d", m.currentPage(), m.totalPages())
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
	if m.mode == modeVaultPicker {
		body = renderPanel(m.styles.panel, screenW, m.bodyHeight(), m.vaultPickerView())
	} else if m.mode == modeVaultName {
		body = m.vaultNameView()
	} else if m.mode == modeVault {
		body = renderPanel(m.styles.panel, screenW, m.bodyHeight(), "Encrypted markdown vault\n\n"+m.password.View())
	} else if m.mode == modeVaultConfirm {
		body = renderPanel(m.styles.panel, screenW, m.bodyHeight(), "Confirm encrypted markdown vault\n\n"+m.confirmPass.View())
	} else if m.mode == modeAI {
		body = m.aiPromptView()
	} else if m.mode == modeGenerating {
		body = m.generatingView()
	} else if m.mode == modeSaveFile {
		body = m.saveFileView()
	} else if m.mode == modeSaveVault {
		body = m.saveVaultView()
	} else if m.mode == modeNewFolder {
		body = m.newFolderView()
	} else if m.mode == modeConfirmDelete {
		body = m.confirmDeleteView()
	} else if m.mode == modeConfirmEyesOff {
		body = m.confirmEyesOffView()
	} else if m.mode == modeRenameTree {
		body = m.renameTreeView()
	} else if m.mode == modeHelp {
		body = m.helpScreenView()
	} else if m.mode == modeFind {
		body = m.findView()
	} else if m.mode == modeJumpPage {
		body = m.jumpPageView()
	} else if m.mode == modeImporting {
		body = m.importingView()
	} else {
		body = m.writeView()
	}
	body = lipgloss.NewStyle().MaxWidth(screenW).MaxHeight(m.bodyHeight()).Render(body)
	help := m.styles.help.Inline(true).MaxWidth(screenW).Render(m.helpText())
	out := strings.Join([]string{header, status, body, help}, "\n")
	return m.styles.frame.Width(screenW).Height(screenH).MaxWidth(screenW).MaxHeight(screenH).Render(out)
}

func (m model) vaultPickerView() string {
	if len(m.vaults) == 0 {
		return "Select vault\n\nNo vaults found.\n\nPress n to create a vault."
	}
	var b strings.Builder
	b.WriteString("Select vault\n\n")
	for i, vault := range m.vaults {
		detail := vault.name
		if !vault.exists {
			detail += " (new default)"
		}
		line := "  " + detail
		if i == m.vaultIdx {
			line = m.styles.sidebarSel.Render("> " + detail)
		}
		b.WriteString(line)
		b.WriteByte('\n')
		pathLine := "    " + vault.path
		if i == m.vaultIdx {
			pathLine = m.styles.sidebarDim.Render(pathLine)
		}
		b.WriteString(pathLine)
		if i != len(m.vaults)-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.help.Render("enter selects, n creates a new vault"))
	return b.String()
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
	if m.selectionMode {
		mainContent = m.selectionView(contentWidth(m.styles.activePanel, mainW), contentHeight(m.styles.activePanel, innerH))
	}
	main := renderPanel(m.styles.activePanel, mainW, innerH, mainContent)
	if !m.treeVisible {
		return main
	}
	tree := renderPanel(treeStyle, treeW, innerH, m.treeView(contentWidth(treeStyle, treeW), contentHeight(treeStyle, innerH)))
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, main)
}

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

func (m model) vaultNameView() string {
	w := max(20, m.width)
	popupWidth := min(64, max(30, w-4))
	copy := "New vault name:\n\n" + m.vaultName.View()
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
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

func (m model) importingView() string {
	w := max(20, m.width)
	popupWidth := min(72, max(30, w-4))
	copy := fmt.Sprintf("%s importing files into the vault", m.working.View())
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

func (m model) newFolderView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	copy := "New folder:\n\n" + m.folderPrompt.View()
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) confirmDeleteView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	target := m.deleteTarget.path
	if target == "" {
		target = m.deleteTarget.name
	}
	copy := "Delete?\n\n" + target + "\n\n" + m.styles.help.Render("enter/y confirms, esc/n cancels")
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonViolet).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) confirmEyesOffView() string {
	w := max(20, m.width)
	popupWidth := min(82, max(32, w-4))
	target := m.eyesOffTarget.path
	if target == "" {
		target = m.eyesOffTarget.name
	}
	copy := "Disable Eyes Only?\n\n" + target + "\n\n" + m.styles.help.Render("This re-enables terminal selection/copy for the note. enter/y confirms, esc/n cancels.")
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningOrange).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) renameTreeView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	copy := "Rename/move to:\n\n" + m.renamePrompt.View()
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) findView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	copy := "Find:\n\n" + m.findPrompt.View()
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonCyan).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) jumpPageView() string {
	w := max(20, m.width)
	popupWidth := min(52, max(30, w-4))
	copy := fmt.Sprintf("Jump to page:\n\n%s\n\n%s", m.jumpPrompt.View(), m.styles.help.Render(fmt.Sprintf("Current document has %d pages.", m.totalPages())))
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(neonViolet).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) helpScreenView() string {
	innerH := m.bodyHeight()
	m.helpView.Height = contentHeight(m.styles.panel, innerH)
	m.helpView.Width = contentWidth(m.styles.panel, m.width)
	return renderPanel(m.styles.activePanel, m.width, innerH, m.helpView.View())
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
			if m.treeEntryEyesOnly(entry) {
				line = m.styles.sidebarEyes.Render("> " + name)
			} else {
				line = m.styles.sidebarSel.Render("> " + name)
			}
		} else if m.treeEntryEyesOnly(entry) {
			line = m.styles.sidebarEyes.Render(line)
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

func (m model) treeEntryEyesOnly(entry treeEntry) bool {
	return entry.vault && !entry.isDir && m.eyesOnlyPaths[cleanVaultPath(entry.path)]
}

func (m model) treeEntryLabel(entry treeEntry) string {
	indent := strings.Repeat("  ", entry.depth)
	current := entry.id == m.currentTreeID()
	marker := " "
	if m.carryTarget.id != "" && entry.id == m.carryTarget.id {
		marker = ">"
	}
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
	if m.mode == modeVaultPicker {
		return "up/down select | enter open | n new vault | ctrl+c quit"
	}
	if m.mode == modeVaultName {
		return "enter create | esc cancel | ctrl+c quit"
	}
	if m.mode == modeVault {
		return "enter unlock/create | ctrl+c quit"
	}
	if m.mode == modeVaultConfirm {
		return "enter create | esc restart password | ctrl+c quit"
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
	if m.mode == modeNewFolder {
		return "enter create | esc cancel | ctrl+c quit"
	}
	if m.mode == modeConfirmDelete {
		return "enter/y delete | esc/n cancel | ctrl+c quit"
	}
	if m.mode == modeConfirmEyesOff {
		return "enter/y disable eyes only | esc/n cancel | ctrl+c quit"
	}
	if m.mode == modeRenameTree {
		return "enter rename | esc cancel | ctrl+c quit"
	}
	if m.mode == modeHelp {
		return "up/down scroll | pgup/pgdown | esc close | ctrl+c quit"
	}
	if m.mode == modeFind {
		return "enter find next | esc cancel | ctrl+c quit"
	}
	if m.mode == modeJumpPage {
		return "enter jump | esc cancel | ctrl+c quit"
	}
	if m.mode == modeImporting {
		return "importing in background | ctrl+c quit"
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
	mouse := "mouse:on"
	if !m.mouseCapture {
		mouse = "select:on"
	}
	eyes := ""
	if m.eyesOnly {
		eyes = " eyes-only"
	}
	return mode + eyes + " " + tree + " " + mouse + " | tab focus | enter open | ^S " + target + " | ^P AI | alt+O eyes | ^K commands | ^C"
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

func (m *model) renderHelp() {
	if m.helpView.Width <= 0 {
		m.helpView.Width = max(20, contentWidth(m.styles.panel, m.width))
	}
	m.helpView.SetContent(strings.TrimSpace(`
WeazlWrite help

Main writing
  tab                 Move between the editor/render pane and the tree.
  ctrl+e              Edit mode.
  ctrl+r              Render mode.
  ctrl+s              Save to the current target.
  ctrl+v              Save to: encrypted vault.
  alt+f               Save to: filesystem.
  ctrl+f              Find text in the current edit/render pane.
  ctrl+g              Jump to a page in the current edit/render pane.
  ctrl+n              New untitled vault note.
  ctrl+p              AI insert prompt. The generated Markdown block is inserted at the cursor.
  ctrl+o              Show or hide the tree.
  ctrl+y              Toggle mouse capture. Off means terminal drag-selection works for copying text.
  alt+o               Toggle eyes-only for the current vault note. Disabling asks for confirmation.
  ctrl+k              Open this command screen.
  ? or h              Open this command screen.
  ctrl+c              Quit.

Tree
  up/down or k/j      Move selection.
  enter               Open a file/note, or fold/unfold a folder.
  n                   New folder at the selected location.
  d                   Delete the selected file, note, or empty folder.
  r                   Rename or move the selected entry by typing its new path.
  o                   Toggle eyes-only on a vault note. Disabling asks for confirmation.
  space               Pick up the selected file/note; move to a destination folder; space again to drop.
  i                   Import the selected filesystem file/folder into the vault.
  pgup/pgdown         Page through the tree.
  esc                 Put down a picked-up entry and return focus to the writer.

Vault and filesystem
  Vault entries live inside the encrypted SQLite vault at ~/.weazlwrite/vault.
  Files entries are regular files from the current filesystem folder.
  Moving with space/drop stays inside the same side: vault-to-vault or filesystem-to-filesystem.
  To copy content between sides, use Save to: vault/filesystem or import.

Copying text
  Terminal mouse selection and app mouse scrolling compete for the same events.
  Press ctrl+y to turn mouse capture off, then select/copy text from edit or render with your terminal.
  Press ctrl+y again to restore mouse scrolling in the tree and panes.
  Eyes-only vault notes keep mouse capture on so blocks cannot be copied out with terminal selection.

Import
  Select a filesystem .md, .markdown, .txt, .pdf, or .docx file and press i to import it to the vault.
  Select a filesystem folder and press i to bulk-import it as a vault root.
  Folder imports preserve relative paths, skip hidden folders/files, and are meant for Obsidian-style vaults.
  PDF and DOCX imports are converted to pure Markdown before they are encrypted and saved.
  Image-only PDFs or image-only Word docs cannot be imported because there is no selectable text to convert.
  Existing vault paths are updated in place.

Prompts
  enter confirms.
  esc cancels.
`))
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

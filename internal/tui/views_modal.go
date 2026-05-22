package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) vaultNameView() string {
	return m.modalView("New vault name", m.vaultName.View(), "", neonCyan, 64)
}

func (m model) aiPromptView() string {
	return m.modalView("AI insert prompt", m.aiPrompt.View(), "The generated Markdown block will be inserted at the editor cursor.", neonCyan, 76)
}

func (m model) generatingView() string {
	return m.modalView("AI generating", fmt.Sprintf("%s %s", m.working.View(), m.thinkingPhrase()), "", neonViolet, 60)
}

func (m model) importingView() string {
	return m.modalView("Vault import", fmt.Sprintf("%s importing files into the vault", m.working.View()), "", neonViolet, 72)
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
	return m.modalView("Save to filesystem", m.filePrompt.View(), "", neonCyan, 80)
}

func (m model) saveVaultView() string {
	return m.modalView("Save to encrypted vault", m.vaultPrompt.View(), "", neonViolet, 80)
}

func (m model) newFolderView() string {
	return m.modalView("New folder", m.folderPrompt.View(), "", neonCyan, 80)
}

func (m model) newDocumentView() string {
	return m.modalView("New document", m.renamePrompt.View(), "", neonCyan, 80)
}

func (m model) confirmDeleteView() string {
	w := max(20, m.width)
	popupWidth := min(80, max(30, w-4))
	target := m.deleteTarget.path
	if target == "" {
		target = m.deleteTarget.name
	}
	copy := m.modalContent("Delete?", target, "enter/y confirms, esc/n cancels", warningOrange, max(1, popupWidth-6))
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningOrange).
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
	copy := m.modalContent("Disable Eyes Only?", target, "This re-enables terminal selection/copy for the note. enter/y confirms, esc/n cancels.", warningOrange, max(1, popupWidth-6))
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningOrange).
		Background(panel).
		Padding(1, 2).
		Width(max(1, popupWidth-6)).
		Render(copy))
}

func (m model) renameTreeView() string {
	return m.modalView("Rename or move", m.renamePrompt.View(), "", neonCyan, 80)
}

func (m model) findView() string {
	return m.modalView("Find", m.findPrompt.View(), "", neonCyan, 80)
}

func (m model) jumpPageView() string {
	return m.modalView("Jump to page", m.jumpPrompt.View(), fmt.Sprintf("Current document has %d pages.", m.totalPages()), neonViolet, 52)
}

func (m model) modalView(title, body, help string, color lipgloss.Color, maxWidth int) string {
	w := max(20, m.width)
	popupWidth := min(maxWidth, max(30, w-4))
	contentWidth := max(1, popupWidth-6)
	copy := m.modalContent(title, body, help, color, contentWidth)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Background(panel).
		Padding(1, 2).
		Width(contentWidth).
		Render(copy))
}

func (m model) modalContent(title, body, help string, color lipgloss.Color, width int) string {
	titleStyle := m.styles.modalTitle.Foreground(color)
	rule := m.styles.modalRule.Render(strings.Repeat("─", max(1, width)))
	parts := []string{
		titleStyle.Render(title),
		rule,
		m.styles.modalBody.Render(body),
	}
	if strings.TrimSpace(help) != "" {
		parts = append(parts, m.styles.help.Render(help))
	}
	return strings.Join(parts, "\n\n")
}

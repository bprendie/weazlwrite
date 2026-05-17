package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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

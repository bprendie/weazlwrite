package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func ansiHeader() string {
	return ` __      __          _______________.__  __      __        .__  __ ________  
/  \    /  \ ____   /  |  \____    /|  |/  \    /  \_______|__|/  |\_____  \ 
\   \/\/   // __ \ /   |  |_/     / |  |\   \/\/   /\_  __ \  \   __\_(__  < 
 \        /\  ___//    ^   /     /_ |  |_\        /  |  | \/  ||  | /       \
  \__/\  /  \___  >____   /_______ \|____/\__/\  /   |__|  |__||__|/______  /
       \/       \/     |__|       \/           \/                         \/ `
}

func renderLogo(logo string, width int) string {
	lines := strings.Split(logo, "\n")
	logoWidth := 0
	for _, line := range lines {
		logoWidth = max(logoWidth, lipgloss.Width(line))
	}
	if width < logoWidth {
		return lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render("weazlwrite")
	}
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = line[:width]
		}
	}
	return lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render(strings.Join(lines, "\n"))
}

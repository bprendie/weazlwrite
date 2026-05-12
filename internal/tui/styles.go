package tui

import "github.com/charmbracelet/lipgloss"

const (
	neonPink   = lipgloss.Color("#FF4FD8")
	neonViolet = lipgloss.Color("#8B5CF6")
	neonCyan   = lipgloss.Color("#00E5FF")
	acidGreen  = lipgloss.Color("#B6FF00")
	amber      = lipgloss.Color("#F8D66D")
	ink        = lipgloss.Color("#E8EAF0")
	muted      = lipgloss.Color("#8A90A2")
	void       = lipgloss.Color("#08080D")
	panel      = lipgloss.Color("#11111A")
	panelAlt   = lipgloss.Color("#171522")
	border     = lipgloss.Color("#3B315C")
)

type styles struct {
	frame       lipgloss.Style
	header      lipgloss.Style
	panel       lipgloss.Style
	activePanel lipgloss.Style
	status      lipgloss.Style
	help        lipgloss.Style
	sidebar     lipgloss.Style
	sidebarSel  lipgloss.Style
	sidebarDim  lipgloss.Style
	editor      lipgloss.Style
	preview     lipgloss.Style
	error       lipgloss.Style
}

func newStyles() styles {
	return styles{
		frame: lipgloss.NewStyle().
			Foreground(ink).
			Background(void).
			Padding(1, 2),
		header: lipgloss.NewStyle().
			Foreground(neonPink).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(panel).
			Padding(0, 1),
		activePanel: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(neonViolet).
			Background(panel).
			Padding(0, 1),
		status: lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true),
		help: lipgloss.NewStyle().
			Foreground(muted),
		sidebar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(panelAlt).
			Padding(0, 1),
		sidebarSel: lipgloss.NewStyle().
			Foreground(acidGreen).
			Bold(true),
		sidebarDim: lipgloss.NewStyle().
			Foreground(muted),
		editor: lipgloss.NewStyle().
			Foreground(ink).
			Background(panel),
		preview: lipgloss.NewStyle().
			Foreground(ink).
			Background(panel),
		error: lipgloss.NewStyle().
			Foreground(amber).
			Bold(true),
	}
}

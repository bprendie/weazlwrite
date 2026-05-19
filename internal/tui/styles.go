package tui

import "github.com/charmbracelet/lipgloss"

const (
	neonPink      = lipgloss.Color("#FF4FD8")
	neonViolet    = lipgloss.Color("#8B5CF6")
	neonCyan      = lipgloss.Color("#00E5FF")
	acidGreen     = lipgloss.Color("#B6FF00")
	warningOrange = lipgloss.Color("#FF9F1C")
	amber         = lipgloss.Color("#F8D66D")
	ink           = lipgloss.Color("#E8EAF0")
	muted         = lipgloss.Color("#8A90A2")
	void          = lipgloss.Color("#08080D")
	panel         = lipgloss.Color("#11111A")
	panelAlt      = lipgloss.Color("#171522")
	border        = lipgloss.Color("#3B315C")
)

type styles struct {
	frame       lipgloss.Style
	header      lipgloss.Style
	panel       lipgloss.Style
	activePanel lipgloss.Style
	status      lipgloss.Style
	statusBar   lipgloss.Style
	statusMode  lipgloss.Style
	statusDirty lipgloss.Style
	statusPath  lipgloss.Style
	help        lipgloss.Style
	helpKey     lipgloss.Style
	sidebar     lipgloss.Style
	sidebarSel  lipgloss.Style
	sidebarDim  lipgloss.Style
	sidebarEyes lipgloss.Style
	treeRoot    lipgloss.Style
	treeFolder  lipgloss.Style
	treeFile    lipgloss.Style
	editor      lipgloss.Style
	preview     lipgloss.Style
	error       lipgloss.Style
}

func newStyles() styles {
	return styles{
		frame: lipgloss.NewStyle().
			Foreground(ink).
			Background(void),
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
		statusBar: lipgloss.NewStyle().
			Foreground(ink).
			Background(panelAlt),
		statusMode: lipgloss.NewStyle().
			Foreground(void).
			Background(neonCyan).
			Bold(true).
			Padding(0, 1),
		statusDirty: lipgloss.NewStyle().
			Foreground(void).
			Background(warningOrange).
			Bold(true).
			Padding(0, 1),
		statusPath: lipgloss.NewStyle().
			Foreground(neonViolet).
			Bold(true),
		help: lipgloss.NewStyle().
			Foreground(muted),
		helpKey: lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true),
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
		sidebarEyes: lipgloss.NewStyle().
			Foreground(warningOrange).
			Bold(true),
		treeRoot: lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true),
		treeFolder: lipgloss.NewStyle().
			Foreground(neonViolet),
		treeFile: lipgloss.NewStyle().
			Foreground(ink),
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

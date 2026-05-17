package tui

import tea "github.com/charmbracelet/bubbletea"

func (m model) startHelp() (tea.Model, tea.Cmd) {
	m.mode = modeHelp
	m.editor.Blur()
	m.renderHelp()
	m.status = "help"
	return m, nil
}

func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "h", "?":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "private markdown vault"
		return m, nil
	default:
		var cmd tea.Cmd
		m.helpView, cmd = m.helpView.Update(msg)
		return m, cmd
	}
}

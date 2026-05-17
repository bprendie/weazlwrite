package tui

import tea "github.com/charmbracelet/bubbletea"

func (m model) updateWrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.cycleFocus()
		return m, nil
	case "ctrl+s":
		m.save()
		m.renderTree()
		return m, nil
	case "ctrl+v":
		return m.startSaveVault()
	case "alt+f":
		return m.startSaveFile()
	case "ctrl+f":
		return m.startFind()
	case "ctrl+g":
		return m.startJumpPage()
	case "alt+o":
		return m.toggleCurrentEyesOnly()
	case "ctrl+n":
		m.newVaultNote()
		return m, nil
	case "ctrl+p", "alt+i":
		return m.startAIInsert()
	case "ctrl+o":
		m.toggleTree()
		return m, nil
	case "ctrl+y":
		return m.toggleMouseCapture()
	case "ctrl+k", "?", "h", "f1":
		return m.startHelp()
	case "ctrl+e":
		m.setView(viewEdit)
		return m, nil
	case "ctrl+r":
		m.setView(viewRender)
		return m, nil
	case "pgup":
		if m.focus == focusTree {
			m.pageTree(-1)
			return m, nil
		}
		if m.focus == focusPreview {
			m.preview.PageUp()
			return m, nil
		}
		if m.focus == focusEditor {
			m.editorPageUp()
			return m, nil
		}
	case "pgdown":
		if m.focus == focusTree {
			m.pageTree(1)
			return m, nil
		}
		if m.focus == focusPreview {
			m.preview.PageDown()
			return m, nil
		}
		if m.focus == focusEditor {
			m.editorPageDown()
			return m, nil
		}
	case "home":
		if m.focus == focusPreview {
			m.preview.GotoTop()
			return m, nil
		}
	case "end":
		if m.focus == focusPreview {
			m.preview.GotoBottom()
			return m, nil
		}
	case "esc":
		if m.selectionMode {
			m.selectionMode = false
			m.selecting = false
			m.status = "selection mode off"
			m.resize()
			return m, tea.ClearScreen
		}
		m.setMainFocus()
		return m, nil
	}

	if m.focus == focusTree {
		return m.updateTree(msg)
	}
	if m.view == viewEdit && m.focus == focusEditor {
		if m.selectionMode {
			return m, nil
		}
		before := m.editor.Value()
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		if m.editor.Value() != before {
			m.dirty = true
			m.renderPreview()
		}
		return m, cmd
	}
	if m.view == viewRender {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	mouse := tea.MouseEvent(msg)
	if m.selectionMode {
		return m.updateSelectionMouse(mouse)
	}
	if msg.Action == tea.MouseActionPress && !mouse.IsWheel() {
		m.setFocus(m.focusAtX(msg.X))
		return m, nil
	}
	if !mouse.IsWheel() {
		return m, nil
	}

	target := m.focusAtX(msg.X)
	if target == focusTree {
		switch msg.Type {
		case tea.MouseWheelUp:
			m.scrollTree(-3)
		case tea.MouseWheelDown:
			m.scrollTree(3)
		}
		return m, nil
	}
	if target == focusEditor && m.view == viewEdit {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	if target == focusPreview {
		switch msg.Type {
		case tea.MouseWheelUp:
			m.preview.ScrollUp(3)
		case tea.MouseWheelDown:
			m.preview.ScrollDown(3)
		default:
			var cmd tea.Cmd
			m.preview, cmd = m.preview.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) cycleFocus() {
	if !m.treeVisible {
		m.setMainFocus()
		return
	}
	if m.focus == focusTree {
		m.setMainFocus()
		return
	}
	m.setFocus(focusTree)
}

func (m *model) setFocus(f focus) {
	m.focus = f
	if f == focusEditor {
		m.editor.Focus()
	} else {
		m.editor.Blur()
	}
	m.resize()
}

func (m *model) setMainFocus() {
	if m.view == viewEdit {
		m.setFocus(focusEditor)
		return
	}
	m.setFocus(focusPreview)
}

func (m *model) setView(v viewMode) {
	m.view = v
	m.setMainFocus()
	if v == viewRender {
		m.renderPreview()
	}
}

func (m *model) toggleTree() {
	m.treeVisible = !m.treeVisible
	if m.treeVisible {
		m.setFocus(focusTree)
		return
	}
	m.setMainFocus()
}

func (m model) toggleMouseCapture() (tea.Model, tea.Cmd) {
	if m.eyesOnly {
		m.selectionMode = false
		m.selecting = false
		m.mouseCapture = true
		m.err = "eyes only notes keep copy protection on"
		m.resize()
		return m, tea.Batch(tea.EnableMouseCellMotion, tea.ClearScreen)
	}
	m.selectionMode = !m.selectionMode
	m.selecting = false
	m.mouseCapture = true
	if !m.selectionMode {
		m.status = "selection mode off"
		m.err = ""
		m.resize()
		return m, tea.Batch(tea.EnableMouseCellMotion, tea.ClearScreen)
	}
	m.initSelectionOffset()
	m.status = "selection mode: drag in the writing pane; release copies"
	m.err = ""
	m.resize()
	return m, tea.Batch(tea.EnableMouseCellMotion, tea.ClearScreen)
}

func (m model) focusAtX(x int) focus {
	treeW, _ := m.layoutWidths()
	if m.treeVisible && x < treeW {
		return focusTree
	}
	if m.view == viewEdit {
		return focusEditor
	}
	return focusPreview
}

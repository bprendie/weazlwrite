package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func (m model) updateWrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editorTyping() && !msg.Paste {
		if m.extendEditorSelection(msg.String()) {
			return m, nil
		}
		switch msg.String() {
		case "alt+w":
			m.cycleWritingWidth()
			return m, nil
		case "alt+t":
			m.toggleTypewriter()
			return m, nil
		case "f3", "shift+f3", "f15", "alt+f3":
			m.repeatEditorFind(msg.String() != "f3")
			return m, nil
		case "ctrl+a":
			m.undoOpen = false
			m.dragStart = textPos{}
			lines := strings.Split(m.editorText(), "\n")
			m.dragEnd = textPos{len(lines) - 1, len([]rune(lines[len(lines)-1]))}
			m.editor.SetPosition(textarea.Position{Line: m.dragEnd.line, Column: m.dragEnd.col})
			return m, nil
		case "ctrl+c":
			return m.copyEditorSelection(false)
		case "ctrl+x":
			return m.copyEditorSelection(true)
		case "esc":
			if m.hasTextSelection() {
				m.clearTextSelection()
				return m, nil
			}
		}
	}
	if m.editorTyping() && (msg.Paste || msg.String() == "tab" || msg.String() == "shift+tab" || msg.String() == "ctrl+v") {
		return m.updateEditorInput(msg)
	}
	if editorEditKind(msg) == "" {
		m.undoOpen = false
	}
	switch msg.String() {
	case keyCycleFocus:
		m.cycleFocus()
		return m, nil
	case keySave:
		m.save()
		m.renderTree()
		return m, nil
	case keySaveVault:
		return m.startSaveVault()
	case keySaveDisk:
		return m.startSaveFile()
	case keyFind:
		return m.startFind()
	case keyJumpPage:
		return m.startJumpPage()
	case keyEyes:
		return m.toggleCurrentEyesOnly()
	case keyNewNote:
		m.newVaultNote()
		return m, nil
	case keyNewFolder:
		if m.focus == focusTree {
			return m.startNewFolder()
		}
	case keyAI:
		return m.startAIInsert()
	case keyToggleTree:
		m.toggleTree()
		return m, nil
	case keySelection:
		return m.toggleMouseCapture()
	case keyUndo:
		m.undoEdit()
		return m, nil
	case keyRedo, "ctrl+y", "alt+z":
		m.redoEdit()
		return m, nil
	case keyHelp:
		return m.startHelp()
	case keyLLM:
		return m.startLLMConfig()
	case keyEdit:
		if !m.editorTyping() {
			m.setView(viewEdit)
			return m, nil
		}
	case keyRender:
		m.setView(viewRender)
		return m, nil
	case keyPgUp:
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
	case keyPgDown:
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
	case keyHome:
		if m.focus == focusPreview {
			m.preview.GotoTop()
			return m, nil
		}
	case keyEnd:
		if m.focus == focusPreview {
			m.preview.GotoBottom()
			return m, nil
		}
	case keyEsc:
		if m.selectionMode {
			m.selectionMode = false
			m.selecting = false
			m.status = "selection mode off"
			m.resize()
			return m, tea.ClearScreen
		}
		if m.view == viewRender {
			m.setView(viewEdit)
			return m, nil
		}
		if m.focus == focusEditor {
			m.treeVisible = true
			m.setFocus(focusTree)
			return m, nil
		}
		m.carryTarget = treeEntry{}
		m.setMainFocus()
		return m, nil
	}

	if (msg.String() == keyHelpQuery || msg.String() == keyHelpH) && !m.editorTyping() {
		return m.startHelp()
	}

	if m.focus == focusTree {
		return m.updateTree(msg)
	}
	if m.view == viewEdit && m.focus == focusEditor {
		if m.selectionMode {
			return m, nil
		}
		return m.updateEditorInput(msg)
	}
	if m.view == viewRender {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
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
	m.undoOpen = false
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
	wasEdit := m.view == viewEdit
	m.view = v
	m.setMainFocus()
	if v == viewRender {
		m.renderPreview()
		if wasEdit {
			m.alignPreviewToEditor()
		}
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
		m.editorDrag = false
		m.mouseCapture = true
		m.err = "eyes only notes keep copy protection on"
		m.resize()
		return m, tea.Batch(tea.EnableMouseCellMotion, tea.ClearScreen)
	}
	m.selectionMode = false
	m.selecting = false
	m.editorDrag = false
	m.mouseCapture = !m.mouseCapture
	m.err = ""
	m.resize()
	if m.mouseCapture {
		m.status = "app mouse on"
		return m, tea.Batch(tea.EnableMouseCellMotion, tea.ClearScreen)
	}
	m.status = "terminal mouse selection on; alt+m restores app mouse"
	return m, tea.Batch(tea.DisableMouse, tea.ClearScreen)
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

package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

const editorIndentWidth = 4

type editorPasteMsg struct {
	epoch uint64
	text  string
	err   error
}

func editorEditKind(msg tea.KeyMsg) string {
	if msg.Paste {
		return "paste"
	}
	switch msg.String() {
	case "backspace", "ctrl+h", "delete", "ctrl+d", "ctrl+w", "alt+backspace", "alt+delete", "alt+d", "ctrl+k", "ctrl+u":
		return "delete"
	case "enter", "ctrl+m", "tab", "shift+tab":
		return "structural"
	}
	if msg.Type == tea.KeyRunes && !msg.Alt {
		return "insert"
	}
	return ""
}

func (m model) updateEditorInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+v" && !msg.Paste {
		m.undoOpen = false
		epoch := m.editorEpoch
		return m, func() tea.Msg {
			text, err := clipboard.ReadAll()
			return editorPasteMsg{epoch: epoch, text: text, err: err}
		}
	}
	before := m.editorSnapshot()
	kind := editorEditKind(msg)
	isolated := kind != "insert" && kind != "delete"
	if isolated || kind != m.lastEditKind {
		m.undoOpen = false
	}
	var cmd tea.Cmd
	hadSelection := m.hasTextSelection()
	if hadSelection && (kind != "" || msg.String() == "ctrl+x") {
		m.undoOpen = false
		isolated = true
	}
	switch {
	case hadSelection && (msg.String() == "tab" || msg.String() == "shift+tab") && !msg.Paste:
		m.indentSelection(msg.String() == "shift+tab")
	case hadSelection && kind != "":
		text := ""
		if kind == "insert" || msg.Paste {
			text = string(msg.Runes)
		}
		if msg.String() == "enter" || msg.String() == "ctrl+m" {
			text = "\n"
		}
		m.replaceSelection(text)
	case msg.String() == "left" || msg.String() == "right":
		direction := 1
		if msg.String() == "left" {
			direction = -1
		}
		if hadSelection {
			a, b := m.selectionBounds()
			pos := b
			if direction < 0 {
				pos = a
			}
			m.placeEditorCursor(pos)
			m.clearTextSelection()
		} else {
			m.editor.SetPosition(m.editor.AdjacentPosition(direction))
		}
		m.editor, cmd = m.editor.Update(nil)
	case msg.Paste:
		m.editor.InsertString(hideTabs(string(msg.Runes)))
		m.editor, cmd = m.editor.Update(nil)
	case msg.String() == "tab":
		m.markdownTab()
		m.editor, cmd = m.editor.Update(nil)
	case msg.String() == "enter" || msg.String() == "ctrl+m":
		m.markdownEnter()
		m.editor, cmd = m.editor.Update(nil)
	case (msg.String() == "backspace" || msg.String() == "ctrl+h") && m.markdownBackspace():
		m.undoOpen = false
		isolated = true
		m.editor, cmd = m.editor.Update(nil)
	case msg.String() == "backspace" || msg.String() == "ctrl+h" || msg.String() == "delete" || msg.String() == "ctrl+d":
		direction := 1
		if msg.String() == "backspace" || msg.String() == "ctrl+h" {
			direction = -1
		}
		p := m.editor.AdjacentPosition(direction)
		a, b := m.cursorTextPos(), textPos{p.Line, p.Column}
		if b.less(a) {
			a, b = b, a
		}
		m.replaceTextRange(a, b, "")
		m.editor, cmd = m.editor.Update(nil)
	case msg.String() == "shift+tab":
		m.outdentEditorLine()
		m.editor, cmd = m.editor.Update(nil)
	default:
		if kind == "" {
			m.clearTextSelection()
		}
		m.editor, cmd = m.editor.Update(msg)
	}
	if m.editorText() != before.text {
		m.recordEdit(before)
		m.lastEditKind = kind
		m.dirty = true
		m.err = ""
	}
	if isolated {
		m.undoOpen = false
	}
	return m, cmd
}

func (m *model) outdentEditorLine() {
	lines := strings.Split(m.editorText(), "\n")
	line, col := m.editor.Line(), m.editorCursorCol()
	if line >= len(lines) {
		return
	}
	remove := 0
	if strings.HasPrefix(lines[line], "\t") {
		remove = 1
	} else {
		for remove < len(lines[line]) && remove < editorIndentWidth && lines[line][remove] == ' ' {
			remove++
		}
	}
	if remove == 0 {
		return
	}
	lines[line] = lines[line][remove:]
	m.setEditorText(strings.Join(lines, "\n"))
	m.moveEditorToLine(line)
	m.editor.SetCursor(max(0, col-remove))
}

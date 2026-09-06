package tui

import "time"

const (
	maxUndoSteps = 256
	undoBurst    = 800 * time.Millisecond
)

type editorSnapshot struct {
	text string
	line int
	col  int
}

func (m model) editorSnapshot() editorSnapshot {
	return editorSnapshot{
		text: m.editorText(),
		line: m.editor.Line(),
		col:  m.editorCursorCol(),
	}
}

func (m *model) resetEditorHistory() {
	m.undo = nil
	m.redo = nil
	m.undoOpen = false
	m.lastEdit = time.Time{}
}

func (m *model) loadEditorText(s string) {
	m.clearTextSelection()
	m.search = editorSearchState{}
	m.editorEpoch++
	m.saves = vaultSaveState{version: m.saves.version, timer: m.saves.timer + 1}
	m.setEditorText(s)
	m.resetEditorHistory()
}

func (m *model) recordEdit(before editorSnapshot) {
	now := time.Now()
	if m.undoOpen && !m.lastEdit.IsZero() && now.Sub(m.lastEdit) < undoBurst && before.line == m.lastEditLine {
		m.lastEdit = now
		return
	}
	m.undo = append(m.undo, before)
	if len(m.undo) > maxUndoSteps {
		m.undo = m.undo[len(m.undo)-maxUndoSteps:]
	}
	m.redo = nil
	m.undoOpen = true
	m.lastEdit = now
	m.lastEditLine = before.line
}

func (m *model) restoreSnapshot(snap editorSnapshot) {
	m.clearTextSelection()
	m.setEditorText(snap.text)
	m.moveEditorToLine(snap.line)
	m.editor.SetCursor(snap.col)
	if m.editor.Focused() {
		m.editor, _ = m.editor.Update(nil)
	}
	if m.view == viewRender {
		m.renderPreview()
	}
}

func (m *model) undoEdit() {
	if len(m.undo) == 0 {
		m.status = "nothing to undo"
		return
	}
	cur := m.editorSnapshot()
	prev := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	m.redo = append(m.redo, cur)
	m.undoOpen = false
	m.restoreSnapshot(prev)
	m.dirty = true
	m.status = "undo"
}

func (m *model) redoEdit() {
	if len(m.redo) == 0 {
		m.status = "nothing to redo"
		return
	}
	cur := m.editorSnapshot()
	next := m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	m.undo = append(m.undo, cur)
	m.undoOpen = false
	m.restoreSnapshot(next)
	m.dirty = true
	m.status = "redo"
}

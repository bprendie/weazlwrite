package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
)

// App chords. Values match tea.KeyMsg.String().
const (
	keyCycleFocus = "tab"
	keySave       = "ctrl+s"
	keySaveVault  = "alt+v"
	keySaveDisk   = "alt+d"
	keyFind       = "ctrl+f"
	keyJumpPage   = "ctrl+g"
	keyEyes       = "alt+o"
	keyAI         = "alt+i"
	keyNewNote    = "alt+n"
	keyNewFolder  = "ctrl+n"
	keyToggleTree = "ctrl+o"
	keySelection  = "ctrl+y"
	keyUndo       = "ctrl+z"
	keyRedo       = "ctrl+shift+z"
	keyHelp       = "f1"
	keyHelpQuery  = "?"
	keyHelpH      = "h"
	keyLLM        = "ctrl+l"
	keyEdit       = "ctrl+e"
	keyRender     = "ctrl+r"
	keyEsc        = "esc"
	keyPgUp       = "pgup"
	keyPgDown     = "pgdown"
	keyHome       = "home"
	keyEnd        = "end"
)

func newEditorKeyMap() textarea.KeyMap {
	km := textarea.DefaultKeyMap
	// Find owns ctrl+f globally; keep only the arrow for forward-char.
	km.CharacterForward = key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("right", "character forward"),
	)
	return km
}

func (m model) editorTyping() bool {
	return m.focus == focusEditor && m.view == viewEdit && !m.selectionMode
}

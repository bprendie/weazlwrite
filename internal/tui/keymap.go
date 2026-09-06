package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/charmbracelet/bubbles/key"
)

// App chords. Values match tea.KeyMsg.String().
const (
	keyCycleFocus = "tab"
	keySave       = "ctrl+s"
	keySaveVault  = "alt+v"
	keySaveDisk   = "alt+s"
	keyFind       = "ctrl+f"
	keyJumpPage   = "ctrl+g"
	keyEyes       = "alt+o"
	keyAI         = "alt+i"
	keyNewNote    = "alt+n"
	keyNewFolder  = "ctrl+n"
	keyToggleTree = "ctrl+o"
	keySelection  = "alt+m"
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
	km.WordBackward = key.NewBinding(key.WithKeys("ctrl+left", "alt+left", "alt+b"))
	km.WordForward = key.NewBinding(key.WithKeys("ctrl+right", "alt+right", "alt+f"))
	return km
}

func (m model) editorTyping() bool {
	return m.mode == modeWrite && m.focus == focusEditor && m.view == viewEdit && !m.selectionMode
}

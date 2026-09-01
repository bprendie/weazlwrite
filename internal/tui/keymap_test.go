package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func altKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true}
}

func typingEditor(value string) textarea.Model {
	editor := textarea.New()
	editor.KeyMap = newEditorKeyMap()
	editor.Focus()
	editor.SetValue(value)
	editor.SetCursor(0)
	return editor
}

func TestCtrlVDoesNotSaveVaultWhileTyping(t *testing.T) {
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: typingEditor("hello"),
	}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlV})
	got := updated.(model)
	if got.mode != modeWrite {
		t.Fatalf("ctrl+v mode = %v, want modeWrite", got.mode)
	}
}

func TestAltVStartsSaveVault(t *testing.T) {
	m := model{
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusEditor,
		view:         viewEdit,
		vaultPrompt:  textinput.New(),
		editor:       typingEditor("hello"),
		treeVisible:  true,
		treeExpanded: map[string]bool{"vault:": true, "file:": true},
	}
	updated, _ := m.updateWrite(altKey('v'))
	got := updated.(model)
	if got.mode != modeSaveVault {
		t.Fatalf("alt+v mode = %v, want modeSaveVault", got.mode)
	}
}

func TestAltDStartsSaveFile(t *testing.T) {
	m := model{
		styles:      newStyles(),
		mode:        modeWrite,
		focus:       focusEditor,
		view:        viewEdit,
		filePrompt:  textinput.New(),
		editor:      typingEditor("hello"),
		treeVisible: true,
		cwd:         t.TempDir(),
	}
	updated, _ := m.updateWrite(altKey('d'))
	got := updated.(model)
	if got.mode != modeSaveFile {
		t.Fatalf("alt+d mode = %v, want modeSaveFile", got.mode)
	}
}

func TestCtrlEIsLineEndWhileTyping(t *testing.T) {
	editor := typingEditor("hello")
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: editor}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlE})
	got := updated.(model)
	if got.view != viewEdit {
		t.Fatalf("ctrl+e view = %v, want viewEdit", got.view)
	}
	updated, _ = got.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	got = updated.(model)
	if got.editor.Value() != "helloX" {
		t.Fatalf("ctrl+e then X value = %q, want helloX", got.editor.Value())
	}
}

func TestCtrlEEntersEditFromTree(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusTree, view: viewRender, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlE})
	got := updated.(model)
	if got.view != viewEdit || got.focus != focusEditor {
		t.Fatalf("ctrl+e from tree view/focus = %v/%v, want edit/editor", got.view, got.focus)
	}
}

func TestCtrlPDoesNotOpenAIWhileTyping(t *testing.T) {
	m := model{
		styles:   newStyles(),
		mode:     modeWrite,
		focus:    focusEditor,
		view:     viewEdit,
		aiPrompt: textinput.New(),
		editor:   typingEditor("hello"),
	}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlP})
	got := updated.(model)
	if got.mode != modeWrite {
		t.Fatalf("ctrl+p mode = %v, want modeWrite", got.mode)
	}
}

func TestCtrlNDoesNotCreateNoteWhileTyping(t *testing.T) {
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: typingEditor("keep"),
	}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlN})
	got := updated.(model)
	if got.editor.Value() != "keep" {
		t.Fatalf("ctrl+n editor value = %q, want keep", got.editor.Value())
	}
	if got.dirty {
		t.Fatal("ctrl+n marked editor dirty")
	}
}

func TestAltNCreatesVaultNoteWhileTyping(t *testing.T) {
	m := model{
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusEditor,
		view:         viewEdit,
		editor:       typingEditor("keep"),
		treeExpanded: map[string]bool{"vault:": true, "file:": true},
	}
	updated, _ := m.updateWrite(altKey('n'))
	got := updated.(model)
	if got.editor.Value() == "keep" {
		t.Fatal("alt+n left the previous buffer in place")
	}
	if !got.isVault || !got.dirty {
		t.Fatalf("alt+n isVault/dirty = %v/%v, want true/true", got.isVault, got.dirty)
	}
}

func TestCtrlKDoesNotOpenHelpWhileTyping(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: typingEditor("hello")}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlK})
	got := updated.(model)
	if got.mode != modeWrite {
		t.Fatalf("ctrl+k mode = %v, want modeWrite", got.mode)
	}
}

func TestF1OpensHelpWhileTyping(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: typingEditor("hello")}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyF1})
	got := updated.(model)
	if got.mode != modeHelp {
		t.Fatalf("f1 mode = %v, want modeHelp", got.mode)
	}
}

func TestHelpContentUsesRemappedChords(t *testing.T) {
	got := helpContent()
	for _, want := range []string{keySaveVault, keySaveDisk, keyAI, keyNewNote, keyHelp} {
		if !strings.Contains(got, want) {
			t.Fatalf("help missing %q", want)
		}
	}
	for _, old := range []string{"Save to: encrypted vault."} {
		if !strings.Contains(got, old) {
			t.Fatalf("help missing %q", old)
		}
	}
	if strings.Contains(got, "ctrl+k") {
		t.Fatal("help still documents ctrl+k")
	}
	if strings.Contains(got, "ctrl+p") {
		t.Fatal("help still documents ctrl+p")
	}
}

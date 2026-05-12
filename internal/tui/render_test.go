package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestViewDoesNotExceedTerminalWidth(t *testing.T) {
	for _, size := range []struct {
		width  int
		height int
	}{
		{width: 80, height: 24},
		{width: 100, height: 30},
		{width: 120, height: 40},
	} {
		editor := textarea.New()
		editor.SetValue("# Test\n\nBody")
		m := model{
			styles:  newStyles(),
			mode:    modeWrite,
			focus:   focusEditor,
			width:   size.width,
			height:  size.height,
			editor:  editor,
			preview: viewport.New(0, 0),
			status:  "editing test.md",
			treeExpanded: map[string]bool{
				"vault:": true,
				"file:":  true,
			},
			tree: []treeEntry{
				{name: "Vault", id: "vault:", isDir: true, vault: true},
				{name: "Files", id: "file:", isDir: true},
				{name: "test.md", id: "file:test.md", path: "test.md", depth: 1},
			},
		}
		m.resize()
		m.renderPreview()

		for i, line := range strings.Split(m.View(), "\n") {
			if got := lipgloss.Width(line); got > size.width {
				t.Fatalf("%dx%d line %d width = %d, want <= %d: %q", size.width, size.height, i+1, got, size.width, line)
			}
		}
		if got := lipgloss.Height(m.View()); got > size.height {
			t.Fatalf("%dx%d height = %d, want <= %d", size.width, size.height, got, size.height)
		}
	}
}

func TestFullLogoRendersAtEightyColumns(t *testing.T) {
	got := renderLogo(ansiHeader(), 80)
	if !strings.Contains(got, "_______________.__") {
		t.Fatalf("expected full ASCII logo at 80 columns, got %q", got)
	}
}

func TestTabCyclesPanes(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(model)
	if got.focus != focusTree {
		t.Fatalf("tab from editor focus = %v, want tree", got.focus)
	}
	updated, _ = got.updateWrite(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(model)
	if got.focus != focusEditor {
		t.Fatalf("tab from tree focus = %v, want editor", got.focus)
	}
}

func TestCtrlPOpensAIPrompt(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, aiPrompt: textinput.New(), editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlP})
	got := updated.(model)
	if got.mode != modeAI {
		t.Fatalf("ctrl+p mode = %v, want modeAI", got.mode)
	}
}

func TestCtrlRSwitchesToRenderMode(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlR})
	got := updated.(model)
	if got.view != viewRender || got.focus != focusPreview {
		t.Fatalf("ctrl+r view/focus = %v/%v, want render/preview", got.view, got.focus)
	}
}

func TestTreeSelectionScrollsIntoView(t *testing.T) {
	m := model{
		styles:      newStyles(),
		mode:        modeWrite,
		focus:       focusTree,
		view:        viewEdit,
		treeVisible: true,
		width:       80,
		height:      12,
	}
	for i := 0; i < 30; i++ {
		m.tree = append(m.tree, treeEntry{name: "note.md", id: "vault:note"})
	}
	m.treeIdx = 25
	m.ensureTreeSelectionVisible()
	if m.treeOffset == 0 {
		t.Fatal("tree offset stayed at top for off-screen selection")
	}
	if m.treeIdx < m.treeOffset || m.treeIdx >= m.treeOffset+m.treeContentHeight() {
		t.Fatalf("tree selection %d not visible in offset %d height %d", m.treeIdx, m.treeOffset, m.treeContentHeight())
	}
}

func TestVaultTreeEntriesUseFilesystemStyleHierarchy(t *testing.T) {
	entries := vaultTreeEntries([]string{
		"projects/specs/api.md",
		"projects/readme.md",
		"daily.md",
	}, map[string]bool{
		"vault:":               true,
		"vault:projects":       true,
		"vault:projects/specs": true,
	})
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.id+" "+strings.Repeat("  ", entry.depth)+entry.name)
	}
	want := []string{
		"vault:daily.md   daily.md",
		"vault:projects   projects/",
		"vault:projects/readme.md     readme.md",
		"vault:projects/specs     specs/",
		"vault:projects/specs/api.md       api.md",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("vault tree:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

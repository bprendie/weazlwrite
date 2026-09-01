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

func TestEscapeMovesFromRenderToEdit(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusPreview, view: viewRender, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.view != viewEdit {
		t.Fatalf("esc from render view = %v, want edit", got.view)
	}
	if got.focus != focusEditor {
		t.Fatalf("esc from render focus = %v, want editor", got.focus)
	}
}

func TestEscapeMovesFromEditorToTree(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, treeVisible: true, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.focus != focusTree {
		t.Fatalf("esc from editor focus = %v, want tree", got.focus)
	}
}

func TestAltIOpensAIPrompt(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, aiPrompt: textinput.New(), editor: textarea.New()}
	updated, _ := m.updateWrite(altKey('i'))
	got := updated.(model)
	if got.mode != modeAI {
		t.Fatalf("alt+i mode = %v, want modeAI", got.mode)
	}
}

func TestPlainHTypesInEditorInsteadOfOpeningHelp(t *testing.T) {
	editor := textarea.New()
	editor.Focus()
	m := model{styles: newStyles(), mode: modeWrite, focus: focusEditor, view: viewEdit, editor: editor}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(model)
	if got.mode != modeWrite {
		t.Fatalf("plain h mode = %v, want modeWrite", got.mode)
	}
	if got.editor.Value() != "h" {
		t.Fatalf("editor value = %q, want h", got.editor.Value())
	}
}

func TestPlainHStillOpensHelpOutsideEditorTyping(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusTree, view: viewEdit, editor: textarea.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(model)
	if got.mode != modeHelp {
		t.Fatalf("plain h from tree mode = %v, want modeHelp", got.mode)
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
	}, []string{
		"projects/archive",
	}, map[string]bool{
		"vault:":                 true,
		"vault:projects":         true,
		"vault:projects/archive": true,
		"vault:projects/specs":   true,
	})
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.id+" "+strings.Repeat("  ", entry.depth)+entry.name)
	}
	want := []string{
		"vault:projects   projects/",
		"vault:projects/archive     archive/",
		"vault:projects/specs     specs/",
		"vault:projects/specs/api.md       api.md",
		"vault:projects/readme.md     readme.md",
		"vault:daily.md   daily.md",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("vault tree:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestReadTreePreservesRootsAndEyesOnlyLookup(t *testing.T) {
	entries, err := readTree(t.TempDir(), []string{"private/plan.md"}, []string{"private"}, map[string]bool{
		"vault:":        true,
		"vault:private": true,
		"file:":         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 {
		t.Fatalf("entries = %d, want roots and vault children", len(entries))
	}
	if entries[0].id != "vault:" || !entries[0].vault || !entries[0].isDir {
		t.Fatalf("first entry = %+v, want vault root", entries[0])
	}
	foundFilesRoot := false
	foundEyesOnly := false
	m := model{eyesOnlyPaths: map[string]bool{"private/plan.md": true}}
	for _, entry := range entries {
		if entry.id == "file:" && entry.isDir {
			foundFilesRoot = true
		}
		if entry.id == "vault:private/plan.md" {
			foundEyesOnly = m.treeEntryEyesOnly(entry)
		}
	}
	if !foundFilesRoot {
		t.Fatal("missing filesystem root")
	}
	if !foundEyesOnly {
		t.Fatal("vault note did not resolve as eyes-only")
	}
}

func TestSelectionBoundsExcludeTreeAndSelectedTextExcludesLineNumbers(t *testing.T) {
	editor := textarea.New()
	editor.SetValue("alpha\nbeta\ngamma")
	m := model{
		styles:      newStyles(),
		mode:        modeWrite,
		focus:       focusEditor,
		view:        viewEdit,
		treeVisible: true,
		width:       100,
		height:      24,
		editor:      editor,
		preview:     viewport.New(0, 0),
	}
	m.resize()
	contentX, contentY, _, _ := m.mainContentBounds()
	if _, ok := m.selectionRowAt(1, contentY); ok {
		t.Fatal("selection row accepted x inside tree")
	}
	row, ok := m.selectionRowAt(contentX, contentY+1)
	if !ok {
		t.Fatal("selection row rejected x inside writing pane")
	}
	if row != 1 {
		t.Fatalf("selection row = %d, want 1", row)
	}
	m.selectStart = selectPoint{row: 0}
	m.selectEnd = selectPoint{row: 1}
	if got := m.selectedText(); got != "alpha\nbeta" {
		t.Fatalf("selectedText = %q, want editor content without line numbers", got)
	}
}

func TestSelectionViewWrapsLongLine(t *testing.T) {
	editor := textarea.New()
	long := strings.Repeat("word ", 30)
	editor.SetValue(long)
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: editor,
	}
	m.selectStart = selectPoint{row: 0}
	m.selectEnd = selectPoint{row: 0}
	got := m.selectionView(20, 12)
	if strings.Count(got, "\n") < 1 {
		t.Fatalf("selection view did not wrap, got %q", got)
	}
	if !strings.Contains(got, "word") {
		t.Fatalf("selection view missing content: %q", got)
	}
	if got := m.selectedText(); got != long {
		t.Fatalf("selectedText copied wrapped visual rows")
	}
}

func TestLayoutWidthsForTreeStates(t *testing.T) {
	m := model{width: 100, height: 24, styles: newStyles(), treeVisible: true, focus: focusEditor}
	treeW, mainW := m.layoutWidths()
	if treeW != 25 || mainW != 75 {
		t.Fatalf("compact layout = %d/%d, want 25/75", treeW, mainW)
	}
	m.focus = focusTree
	treeW, mainW = m.layoutWidths()
	if treeW != 55 || mainW != 45 {
		t.Fatalf("focused layout = %d/%d, want 55/45", treeW, mainW)
	}
	m.treeVisible = false
	treeW, mainW = m.layoutWidths()
	if treeW != 0 || mainW != 100 {
		t.Fatalf("hidden tree layout = %d/%d, want 0/100", treeW, mainW)
	}
}

func TestNewVaultNotePathUsesSelectedVaultFolder(t *testing.T) {
	tests := []struct {
		name  string
		tree  []treeEntry
		idx   int
		vault string
		want  string
	}{
		{
			name: "selected folder",
			tree: []treeEntry{
				{name: "Vault", id: "vault:", isDir: true, vault: true},
				{name: "specs/", id: "vault:projects/specs", path: "projects/specs", isDir: true, vault: true},
			},
			idx:  1,
			want: "projects/specs/untitled.md",
		},
		{
			name: "selected note parent",
			tree: []treeEntry{
				{name: "readme.md", id: "vault:projects/readme.md", path: "projects/readme.md", vault: true},
			},
			want: "projects/untitled.md",
		},
		{
			name:  "current vault note parent",
			tree:  []treeEntry{{name: "Files", id: "file:", isDir: true}},
			vault: "daily/monday.md",
			want:  "daily/untitled.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{tree: tt.tree, treeIdx: tt.idx, isVault: tt.vault != "", vaultPath: tt.vault}
			if got := m.newVaultNotePath("untitled.md"); got != tt.want {
				t.Fatalf("newVaultNotePath = %q, want %q", got, tt.want)
			}
		})
	}
}

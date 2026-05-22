package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/weazlwrite/internal/storage"
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

func TestRenderTreeAppliesPersistedVaultFolderCollapse(t *testing.T) {
	store, err := storage.Open(t.TempDir() + "/vault.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateVault("secret"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveNote("id", "projects/specs/api.md", "api", "# API"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetFolderCollapsed("projects/specs", true); err != nil {
		t.Fatal(err)
	}

	m := model{
		store:        store,
		styles:       newStyles(),
		width:        100,
		height:       24,
		treeVisible:  true,
		cwd:          t.TempDir(),
		treeExpanded: map[string]bool{"vault:": true, "file:": true, "vault:projects": true},
	}
	if err := m.renderTree(); err != nil {
		t.Fatal(err)
	}
	if m.treeExpanded["vault:projects/specs"] {
		t.Fatal("persisted collapsed folder rendered as expanded")
	}
	for _, entry := range m.tree {
		if entry.id == "vault:projects/specs/api.md" {
			t.Fatal("note inside collapsed folder should not be visible")
		}
	}
	if err := store.SetFolderCollapsed("projects/specs", false); err != nil {
		t.Fatal(err)
	}
	if err := m.renderTree(); err != nil {
		t.Fatal(err)
	}
	if !m.treeExpanded["vault:projects/specs"] {
		t.Fatal("persisted expanded folder rendered as collapsed")
	}
}

func TestTreeLeftRightExpandCollapseSelectedFolder(t *testing.T) {
	store, err := storage.Open(t.TempDir() + "/vault.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateVault("secret"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveNote("id", "projects/specs/api.md", "api", "# API"); err != nil {
		t.Fatal(err)
	}

	m := model{
		store:        store,
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusTree,
		width:        100,
		height:       24,
		treeVisible:  true,
		cwd:          t.TempDir(),
		treeExpanded: map[string]bool{"vault:": true, "file:": true, "vault:projects": true},
	}
	if err := m.renderTree(); err != nil {
		t.Fatal(err)
	}
	for i, entry := range m.tree {
		if entry.id == "vault:projects/specs" {
			m.treeIdx = i
			break
		}
	}
	if m.tree[m.treeIdx].id != "vault:projects/specs" {
		t.Fatal("missing selected specs folder")
	}

	updated, _ := m.updateTree(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if !m.treeExpanded["vault:projects/specs"] {
		t.Fatal("right arrow did not expand selected folder")
	}

	updated, _ = m.updateTree(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(model)
	if m.treeExpanded["vault:projects/specs"] {
		t.Fatal("left arrow did not collapse selected folder")
	}

	folders, err := store.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if folder.Path == "projects/specs" && !folder.Collapsed {
			t.Fatal("left arrow collapse was not persisted")
		}
	}
}

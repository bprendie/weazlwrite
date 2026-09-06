package tui

import (
	"github.com/bprendie/weazlwrite/internal/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

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

func TestCtrlNFromTreeStartsNewFolder(t *testing.T) {
	m := model{styles: newStyles(), mode: modeWrite, focus: focusTree, view: viewEdit, folderPrompt: textinput.New()}
	updated, _ := m.updateWrite(tea.KeyMsg{Type: tea.KeyCtrlN})
	got := updated.(model)
	if got.mode != modeNewFolder {
		t.Fatalf("ctrl+n from tree mode = %v, want modeNewFolder", got.mode)
	}
}

func TestConfirmDeleteRemovesEmptyVaultFolder(t *testing.T) {
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
	if err := store.SaveFolder("projects/empty"); err != nil {
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
		if entry.id == "vault:projects/empty" {
			m.treeIdx = i
			break
		}
	}
	if m.tree[m.treeIdx].id != "vault:projects/empty" {
		t.Fatal("missing selected empty folder")
	}

	updated, _ := m.startConfirmDelete()
	m = updated.(model)
	updated, _ = m.updateConfirmDelete(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.err != "" {
		t.Fatalf("delete empty vault folder err = %q", m.err)
	}
	for _, entry := range m.tree {
		if entry.id == "vault:projects/empty" {
			t.Fatal("empty vault folder still rendered after delete")
		}
	}
}

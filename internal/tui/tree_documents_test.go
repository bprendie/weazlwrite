package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/bprendie/weazlwrite/internal/storage"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"testing"
)

func TestTreeNCreatesVaultDocumentInSelectedFolder(t *testing.T) {
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
	if err := store.SaveFolder("projects/specs"); err != nil {
		t.Fatal(err)
	}

	m := model{
		store:        store,
		styles:       newStyles(),
		mode:         modeWrite,
		focus:        focusTree,
		view:         viewEdit,
		width:        100,
		height:       24,
		treeVisible:  true,
		cwd:          t.TempDir(),
		treeExpanded: map[string]bool{"vault:": true, "file:": true, "vault:projects": true, "vault:projects/specs": true},
		editor:       textarea.New(),
		renamePrompt: textinput.New(),
		preview:      viewport.New(0, 0),
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

	updated, _ := m.updateTree(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(model)
	if m.mode != modeNewDocument {
		t.Fatalf("n from tree mode = %v, want modeNewDocument", m.mode)
	}
	if got := m.renamePrompt.Value(); got != "projects/specs/" {
		t.Fatalf("new document prompt = %q, want projects/specs/", got)
	}
	m.renamePrompt.SetValue("projects/specs/design")
	updated, _ = m.updateNewDocument(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.vaultPath != "projects/specs/design.md" {
		t.Fatalf("vaultPath = %q, want projects/specs/design.md", m.vaultPath)
	}
	if m.dirty {
		t.Fatal("new tree vault document should be saved immediately")
	}
	if m.focus != focusEditor {
		t.Fatalf("new tree vault document focus = %v, want editor", m.focus)
	}
	if _, _, ok, err := store.LoadNote(m.vaultPath); err != nil || !ok {
		t.Fatalf("created vault note load ok=%v err=%v", ok, err)
	}
}

func TestTreeNCreatesFilesystemDocumentInSelectedFolder(t *testing.T) {
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
	cwd := t.TempDir()
	dir := filepath.Join(cwd, "drafts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := model{
		store:       store,
		styles:      newStyles(),
		mode:        modeWrite,
		focus:       focusTree,
		view:        viewEdit,
		width:       100,
		height:      24,
		treeVisible: true,
		cwd:         cwd,
		tree: []treeEntry{
			{name: "Files", id: "file:", isDir: true},
			{name: "drafts/", id: "file:" + dir, path: dir, isDir: true, depth: 1},
		},
		treeIdx:      1,
		editor:       textarea.New(),
		renamePrompt: textinput.New(),
		preview:      viewport.New(0, 0),
		isVault:      true,
		vaultPath:    "old.md",
	}

	updated, _ := m.updateTree(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(model)
	if m.mode != modeNewDocument {
		t.Fatalf("n from tree mode = %v, want modeNewDocument", m.mode)
	}
	if got := m.renamePrompt.Value(); got != dir+string(filepath.Separator) {
		t.Fatalf("new document prompt = %q, want %s", got, dir+string(filepath.Separator))
	}
	want := filepath.Join(dir, "draft")
	m.renamePrompt.SetValue(want)
	updated, _ = m.updateNewDocument(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.diskPath != want+".md" {
		t.Fatalf("diskPath = %q, want %s.md", m.diskPath, want)
	}
	if m.isVault {
		t.Fatal("new filesystem document should switch out of vault mode")
	}
	if _, err := os.Stat(m.diskPath); err != nil {
		t.Fatalf("created file stat: %v", err)
	}
}

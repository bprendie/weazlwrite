package tui

import (
	"strings"
	"testing"
)

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

package storage

import (
	"path/filepath"
	"testing"
)

func TestScanBoolAcceptsLegacyEyesOnlyValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "zero int64", value: int64(0), want: false},
		{name: "one int64", value: int64(1), want: true},
		{name: "false string", value: "false", want: false},
		{name: "true string", value: "true", want: true},
		{name: "false bytes", value: []byte("false"), want: false},
		{name: "true bytes", value: []byte("true"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scanBool(tt.value); got != tt.want {
				t.Fatalf("scanBool(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestEyesOnlyMigrationAndLoadAcceptSQLiteBoolShapes(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
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

	paths := []string{"one.md", "two.md", "three.md", "four.md"}
	for _, path := range paths {
		if err := store.SaveNote(path, path, path, "body"); err != nil {
			t.Fatalf("save %s: %v", path, err)
		}
	}
	updates := map[string]any{
		"one.md":   0,
		"two.md":   1,
		"three.md": "false",
		"four.md":  "true",
	}
	for path, value := range updates {
		if _, err := store.db.Exec(`update notes set eyes_only = ? where path = ?`, value, path); err != nil {
			t.Fatalf("set %s eyes_only: %v", path, err)
		}
	}
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"one.md":   false,
		"two.md":   true,
		"three.md": false,
		"four.md":  true,
	}
	for path, expected := range want {
		note, _, ok, err := store.LoadNote(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		if !ok {
			t.Fatalf("load %s: not found", path)
		}
		if note.EyesOnly != expected {
			t.Fatalf("%s EyesOnly = %v, want %v", path, note.EyesOnly, expected)
		}
	}
}

func TestFolderCollapsedPersistsThroughListFolders(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
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
	if err := store.SetFolderCollapsed("projects/specs", true); err != nil {
		t.Fatal(err)
	}

	folders, err := store.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	collapsed := map[string]bool{}
	for _, folder := range folders {
		collapsed[folder.Path] = folder.Collapsed
	}
	if collapsed["projects"] {
		t.Fatal("parent folder should default to expanded")
	}
	if !collapsed["projects/specs"] {
		t.Fatal("folder collapsed state was not persisted")
	}
}

func TestDeleteFolderRemovesEmptyFolderSubtree(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
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
	if err := store.SaveFolder("projects/empty/nested"); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteFolder("projects/empty"); err != nil {
		t.Fatal(err)
	}

	folders, err := store.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if folder.Path == "projects/empty" || folder.Path == "projects/empty/nested" {
			t.Fatalf("empty folder subtree still exists: %s", folder.Path)
		}
	}
}

func TestDeleteFolderBlocksWhenNotesExistBelow(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
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
	if err := store.SaveNote("id", "projects/live/note.md", "note", "# Note"); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteFolder("projects/live"); err == nil {
		t.Fatal("expected delete to fail when notes exist below folder")
	}
}

func TestDeleteFolderDoesNotTreatPathCharactersAsWildcards(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
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
	if err := store.SaveFolder("work_a/empty"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveNote("id", "workXa/note.md", "note", "# Note"); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteFolder("work_a/empty"); err != nil {
		t.Fatal(err)
	}

	folders, err := store.ListFolders()
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if folder.Path == "work_a/empty" {
			t.Fatal("folder with underscore path was not deleted")
		}
	}
}

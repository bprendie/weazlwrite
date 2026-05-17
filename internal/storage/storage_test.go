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

package storage

import (
	"path/filepath"
	"testing"
)

func TestDraftNamesPersistWithoutOverwritingOrDuplicating(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "vault.sqlite3")
	s, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateVault("test-password"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveNote("existing", "folder/hello.md", "hello", "original"); err != nil {
		t.Fatal(err)
	}
	chosen, err := s.SaveDraft("draft", "folder/hello.md", "hello", "draft text", true)
	if err != nil || chosen != "folder/hello-2.md" {
		t.Fatalf("collision: %s %v", chosen, err)
	}
	if err := s.SetNoteEyesOnly(chosen, true); err != nil {
		t.Fatal(err)
	}
	chosen, err = s.SaveDraft("draft", "folder/hello-world.md", "hello world", "updated draft", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveDraft("draft", "folder/hello.md", "hello", "oops", false); err == nil {
		t.Fatal("explicit name overwrote another note")
	}
	notes, err := s.ListNotes()
	if err != nil || len(notes) != 2 {
		t.Fatalf("duplicated notes: %v %v", notes, err)
	}
	s.Close()
	s, err = Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err = s.Unlock("test-password"); err != nil {
		t.Fatal(err)
	}
	note, content, ok, err := s.LoadNote(chosen)
	if err != nil || !ok || content != "updated draft" || !note.AutoNamed || !note.EyesOnly {
		t.Fatalf("reopen: %+v %q %v", note, content, err)
	}
	chosen, err = s.SaveDraft("draft", "folder/final.md", "final", content, false)
	if err != nil {
		t.Fatal(err)
	}
	note, _, _, _ = s.LoadNote(chosen)
	if note.AutoNamed || !note.EyesOnly {
		t.Fatal("confirmation metadata")
	}
	_, content, _, _ = s.LoadNote("folder/hello.md")
	if content != "original" {
		t.Fatal("existing content modified")
	}
}

func TestDraftNameMigrationPreservesExistingNamedNotes(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "vault.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateVault("test-password"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveNote("old", "old.md", "old", "old content"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`alter table notes drop column auto_named`); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	n, content, ok, err := s.LoadNote("old.md")
	if err != nil || !ok || n.AutoNamed || content != "old content" {
		t.Fatalf("migration: %+v %q %v", n, content, err)
	}
}

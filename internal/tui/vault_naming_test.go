package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestReadableVaultNames(t *testing.T) {
	cases := map[string]string{
		"# Untitled\nFirst few words of a really long document":              "first-few-words-of-a-really.md",
		"# **Project** [roadmap](https://example.com/private) &amp; `notes`": "project-roadmap-notes.md",
		"---\ntitle: hidden metadata\n---\n> 1. Actual writing here":         "actual-writing-here.md",
		"- [x] Ship the thing":                         "ship-the-thing.md",
		"```go\nsecretCode()\n```\nUseful explanation": "useful-explanation.md",
		"```\ncode only\n```":                          "untitled.md",
		"日本語の文章 café résumé":                           "日本語の文章-café-résumé.md",
		"../../bad / filename:*?":                      "bad-filename.md",
		"# Untitled\n":                                 "untitled.md",
	}
	for content, want := range cases {
		if got := suggestedVaultName(content); got != want {
			t.Errorf("%q: %q want %q", content, got, want)
		}
	}
	if len([]rune(suggestedVaultName(strings.Repeat("界", 100)))) > 63 {
		t.Fatal("name too long")
	}
}

func namedDraftModel(t *testing.T) model {
	m := vaultModel(t)
	m.vaultPrompt = textinput.New()
	m.newVaultNote()
	m, _ = updateModel(m, textKey("A readable draft"))
	return m
}

func TestFirstSaveNamesDraftAfterAutosaveAndReopen(t *testing.T) {
	m := namedDraftModel(t)
	cmd := m.startVaultSave(false)
	m, _ = updateModel(m, cmd())
	if m.vaultPath != "a-readable-draft.md" || !m.autoNamed {
		t.Fatalf("automatic path: %s", m.vaultPath)
	}
	id := m.vaultID
	if err := m.openVaultPath(m.vaultPath); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeSaveVault || m.vaultPrompt.Value() != "a-readable-draft.md" {
		t.Fatal("first save did not suggest name")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.autoNamed || persisted(t, m, m.vaultPath) != "# Untitled\nA readable draft" {
		t.Fatal("cancel lost draft")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	m.vaultPrompt.SetValue("my-final-name")
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeWrite || m.vaultPath != "my-final-name.md" || m.autoNamed {
		t.Fatalf("accept: %s %s", m.vaultPath, m.err)
	}
	notes, _ := m.store.ListNotes()
	if len(notes) != 1 || notes[0].ID != id || notes[0].AutoNamed {
		t.Fatal("rename duplicated draft or lost identity")
	}
	if err := m.openVaultPath(m.vaultPath); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeWrite {
		t.Fatal("confirmed name prompted again")
	}
}

func TestFirstSaveDrainsOldWriteAndProtectsCollision(t *testing.T) {
	m := namedDraftModel(t)
	old := m.startVaultSave(false)
	m, _ = updateModel(m, textKey(" updated"))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeWrite || m.saves.pending == nil {
		t.Fatal("prompt raced old write")
	}
	m, next := updateModel(m, old())
	m, _ = updateModel(m, next())
	if m.mode != modeSaveVault || m.vaultPrompt.Value() != "a-readable-draft-updated.md" {
		t.Fatal("stale suggestion")
	}
	if err := m.store.SaveNote("other", "taken.md", "taken", "keep me"); err != nil {
		t.Fatal(err)
	}
	m.vaultPrompt.SetValue("taken.md")
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeSaveVault || m.err == "" || persisted(t, m, "taken.md") != "keep me" {
		t.Fatal("collision not protected")
	}
}

func TestDraftNamingFailureKeepsWriting(t *testing.T) {
	m := namedDraftModel(t)
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("missing write")
	}
	m, _ = updateModel(m, vaultSaveResult{snapshot: m.saves.inFlight, err: errors.New("disk full")})
	if !m.dirty || m.mode != modeWrite || m.saves.pending != nil {
		t.Fatal("failed save lost draft or trapped input")
	}
}

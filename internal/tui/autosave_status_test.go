package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBackgroundSaveKeepsStatusQuiet(t *testing.T) {
	m := vaultModel(t)
	m.status = "writing"
	before := m.statusView(200)
	m, _ = updateModel(m, textKey("draft"))
	if got := m.statusView(200); got != before {
		t.Fatalf("typing changed status: %q", got)
	}
	cmd := m.startVaultSave(false)
	if got := m.statusView(200); got != before {
		t.Fatalf("save start changed status: %q", got)
	}
	m, _ = updateModel(m, cmd())
	if got := m.statusView(200); got != before {
		t.Fatalf("save completion changed status: %q", got)
	}
	if got := persisted(t, m, m.vaultPath); got != "draft" {
		t.Fatal(got)
	}
}

func TestExplicitSaveConfirmsLatestDraft(t *testing.T) {
	m := vaultModel(t)
	m.status = "writing"
	m, _ = updateModel(m, textKey("first"))
	old := m.startVaultSave(false)
	m, _ = updateModel(m, textKey(" latest"))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	m, next := updateModel(m, old())
	if strings.Contains(m.status, "saved vault:") {
		t.Fatal("confirmed stale write")
	}
	if next == nil {
		t.Fatal("latest draft was not flushed")
	}
	m, _ = updateModel(m, next())
	if m.status != "saved vault:"+m.vaultPath || m.dirty {
		t.Fatal("missing save confirmation")
	}
	m.status = "writing"
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.status != "saved vault:"+m.vaultPath {
		t.Fatal("clean save not confirmed")
	}
}

func TestQuietSaveStillShowsErrorsAndUnsavedDisk(t *testing.T) {
	m := vaultModel(t)
	m, _ = updateModel(m, textKey("draft"))
	m.startVaultSave(false)
	m, _ = updateModel(m, vaultSaveResult{snapshot: m.saves.inFlight, err: errors.New("offline")})
	if !strings.Contains(m.statusView(200), "save failed") {
		t.Fatal("save failure hidden")
	}
	m.saves.err = ""
	m.isVault = false
	if !strings.Contains(m.statusView(200), "*") {
		t.Fatal("disk dirty marker hidden")
	}
	m.isVault = true
	m.vaultPath = ""
	if !strings.Contains(m.statusView(200), "*") {
		t.Fatal("detached draft marker hidden")
	}
}

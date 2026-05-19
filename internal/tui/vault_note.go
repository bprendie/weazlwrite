package tui

import (
	"strings"

	"github.com/google/uuid"
)

func (m *model) newVaultNote() {
	name := "untitled-" + uuid.NewString()[:8] + ".md"
	path := m.newVaultNotePath(name)
	m.vaultID = uuid.NewString()
	m.filePath = path
	m.vaultPath = path
	m.diskPath = ""
	m.isVault = true
	m.eyesOnly = false
	m.expandTreeTo("vault:" + path)
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + path
	m.setView(viewEdit)
	m.renderPreview()
}

func (m model) newVaultNotePath(name string) string {
	base := m.selectedVaultFolderBase()
	if base == "" && m.isVault && m.vaultPath != "" {
		if parent := vaultParent(m.vaultPath); parent != "" {
			base = parent + "/"
		}
	}
	return cleanVaultPath(base + name)
}

func (m model) selectedVaultFolderBase() string {
	entry, ok := m.selectedTreeEntry()
	if !ok || !entry.vault {
		return ""
	}
	if entry.id == "vault:" {
		return ""
	}
	if entry.isDir {
		return strings.TrimSuffix(entry.path, "/") + "/"
	}
	parent := vaultParent(entry.path)
	if parent == "" {
		return ""
	}
	return parent + "/"
}

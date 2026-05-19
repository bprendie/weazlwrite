package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) treeView(width, height int) string {
	if len(m.tree) == 0 {
		return m.styles.sidebarDim.Render("empty")
	}
	var b strings.Builder
	start := min(max(0, m.treeOffset), max(0, len(m.tree)-1))
	for row, i := 0, start; i < len(m.tree); row, i = row+1, i+1 {
		if row >= height {
			break
		}
		entry := m.tree[i]
		name := m.treeEntryLabel(entry)
		name = minString(name, max(1, width-2))
		line := m.treeEntryStyle(entry, false).Render("  " + name)
		if i == m.treeIdx {
			if m.treeEntryEyesOnly(entry) {
				line = m.styles.sidebarEyes.Render("> " + name)
			} else {
				line = m.styles.sidebarSel.Render("> " + name)
			}
		} else if m.treeEntryEyesOnly(entry) {
			line = m.styles.sidebarEyes.Render("  " + name)
		} else if entry.isDir {
			line = m.treeEntryStyle(entry, false).Render("  " + name)
		}
		b.WriteString(line)
		if i != len(m.tree)-1 && row != height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) treeEntryStyle(entry treeEntry, selected bool) lipgloss.Style {
	if selected {
		return m.styles.sidebarSel
	}
	if entry.id == "vault:" || entry.id == "file:" {
		return m.styles.treeRoot
	}
	if entry.isDir {
		return m.styles.treeFolder
	}
	return m.styles.treeFile
}

func (m model) treeEntryEyesOnly(entry treeEntry) bool {
	return entry.vault && !entry.isDir && m.eyesOnlyPaths[cleanVaultPath(entry.path)]
}

func (m model) treeEntryLabel(entry treeEntry) string {
	indent := strings.Repeat("  ", entry.depth)
	current := entry.id == m.currentTreeID()
	marker := " "
	if m.carryTarget.id != "" && entry.id == m.carryTarget.id {
		marker = ">"
	}
	if current {
		marker = "•"
		if m.dirty {
			marker = "*"
		}
	}
	if entry.isDir {
		icon := "▸"
		if m.treeExpanded[entry.id] {
			icon = "▾"
		}
		return indent + icon + " " + entry.name
	}
	return indent + marker + " " + entry.name
}

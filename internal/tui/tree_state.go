package tui

import (
	"path/filepath"
	"strings"
)

func (m *model) renderTree() error {
	m.ensureTreeExpanded()
	selectedID := ""
	if len(m.tree) > 0 && m.treeIdx < len(m.tree) {
		selectedID = m.tree[m.treeIdx].id
	}
	if selectedID == "" {
		selectedID = m.currentTreeID()
	}
	notes, err := m.store.ListNotes()
	if err != nil {
		return err
	}
	vaultNotes := make([]string, 0, len(notes))
	m.eyesOnlyPaths = map[string]bool{}
	for _, note := range notes {
		vaultNotes = append(vaultNotes, note.Path)
		if note.EyesOnly {
			m.eyesOnlyPaths[cleanVaultPath(note.Path)] = true
		}
	}
	folders, err := m.store.ListFolders()
	if err != nil {
		return err
	}
	vaultFolders := make([]string, 0, len(folders))
	for _, folder := range folders {
		vaultFolders = append(vaultFolders, folder.Path)
	}
	tree, err := readTree(m.cwd, vaultNotes, vaultFolders, m.treeExpanded)
	m.tree = tree
	m.selectTreeID(selectedID)
	return err
}

func (m *model) ensureTreeExpanded() {
	if m.treeExpanded == nil {
		m.treeExpanded = map[string]bool{}
	}
	if _, ok := m.treeExpanded["vault:"]; !ok {
		m.treeExpanded["vault:"] = true
	}
	if _, ok := m.treeExpanded["file:"]; !ok {
		m.treeExpanded["file:"] = true
	}
}

func (m *model) selectTreeID(id string) {
	if id != "" {
		for i, entry := range m.tree {
			if entry.id == id {
				m.treeIdx = i
				m.ensureTreeSelectionVisible()
				return
			}
		}
	}
	if current := m.currentTreeID(); current != "" {
		for i, entry := range m.tree {
			if entry.id == current {
				m.treeIdx = i
				m.ensureTreeSelectionVisible()
				return
			}
		}
	}
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = max(0, len(m.tree)-1)
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) ensureTreeSelectionVisible() {
	if len(m.tree) == 0 {
		m.treeIdx = 0
		m.treeOffset = 0
		return
	}
	if m.treeIdx < 0 {
		m.treeIdx = 0
	}
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = len(m.tree) - 1
	}
	height := m.treeContentHeight()
	if height <= 0 {
		height = 1
	}
	if m.treeIdx < m.treeOffset {
		m.treeOffset = m.treeIdx
	}
	if m.treeIdx >= m.treeOffset+height {
		m.treeOffset = m.treeIdx - height + 1
	}
	maxOffset := max(0, len(m.tree)-height)
	if m.treeOffset > maxOffset {
		m.treeOffset = maxOffset
	}
	if m.treeOffset < 0 {
		m.treeOffset = 0
	}
}

func (m model) treeContentHeight() int {
	treeW, _ := m.layoutWidths()
	if treeW <= 0 {
		return 0
	}
	return contentHeight(m.styles.sidebar, m.bodyHeight())
}

func (m model) currentTreeID() string {
	if m.isVault && m.vaultPath != "" {
		return "vault:" + cleanVaultPath(m.vaultPath)
	}
	if !m.isVault && m.diskPath != "" {
		return "file:" + m.diskPath
	}
	return ""
}

func (m *model) expandTreeTo(id string) {
	m.ensureTreeExpanded()
	switch {
	case id == "vault:" || strings.HasPrefix(id, "vault:"):
		m.treeExpanded["vault:"] = true
		path := strings.TrimPrefix(id, "vault:")
		parts := strings.Split(cleanVaultPath(path), "/")
		var prefix []string
		for i := 0; i < len(parts)-1; i++ {
			prefix = append(prefix, parts[i])
			m.treeExpanded["vault:"+strings.Join(prefix, "/")] = true
		}
	case id == "file:" || strings.HasPrefix(id, "file:"):
		m.treeExpanded["file:"] = true
		path := strings.TrimPrefix(id, "file:")
		dir := filepath.Dir(path)
		for dir != "." && dir != string(filepath.Separator) && strings.HasPrefix(dir, m.cwd) {
			m.treeExpanded["file:"+dir] = true
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
}

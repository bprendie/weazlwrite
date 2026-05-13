package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type treeEntry struct {
	name  string
	path  string
	id    string
	isDir bool
	vault bool
	depth int
}

func readTree(dir string, vaultNotes []string, vaultFolders []string, expanded map[string]bool) ([]treeEntry, error) {
	var entries []treeEntry
	entries = append(entries, treeEntry{name: "Vault", id: "vault:", isDir: true, vault: true})
	if expanded["vault:"] {
		entries = append(entries, vaultTreeEntries(vaultNotes, vaultFolders, expanded)...)
	}

	entries = append(entries, treeEntry{name: "Files", path: dir, id: "file:", isDir: true})
	if expanded["file:"] {
		fileEntries, err := fileTreeEntries(dir, 1, expanded)
		entries = append(entries, fileEntries...)
		if err != nil {
			return entries, err
		}
	}
	return entries, nil
}

func fileTreeEntries(dir string, depth int, expanded map[string]bool) ([]treeEntry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var entries []treeEntry
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir() != items[j].IsDir() {
			return items[i].IsDir()
		}
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if item.IsDir() {
			id := "file:" + path
			entries = append(entries, treeEntry{name: name + "/", path: path, id: id, isDir: true, depth: depth})
			if expanded[id] {
				children, err := fileTreeEntries(path, depth+1, expanded)
				if err != nil {
					return entries, err
				}
				entries = append(entries, children...)
			}
			continue
		}
		if isTreeFile(name) {
			entries = append(entries, treeEntry{name: name, path: path, id: "file:" + path, depth: depth})
		}
	}
	return entries, nil
}

func isTreeFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".txt", ".pdf", ".docx", ".doc":
		return true
	default:
		return false
	}
}

func vaultTreeEntries(notes []string, folders []string, expanded map[string]bool) []treeEntry {
	noteSet := map[string]bool{}
	folderSet := map[string]bool{}
	for _, note := range notes {
		clean := cleanVaultPath(note)
		if clean == "" {
			continue
		}
		noteSet[clean] = true
		addVaultParents(clean, folderSet)
	}
	for _, folder := range folders {
		clean := cleanVaultPath(folder)
		if clean == "" {
			continue
		}
		folderSet[clean] = true
		addVaultParents(clean, folderSet)
	}
	return vaultTreeChildren("", 1, noteSet, folderSet, expanded)
}

func vaultTreeChildren(parent string, depth int, notes, folders map[string]bool, expanded map[string]bool) []treeEntry {
	dirSet := map[string]string{}
	var files []string
	for folder := range folders {
		if vaultParent(folder) == parent {
			dirSet[vaultBase(folder)] = folder
		}
	}
	for note := range notes {
		if vaultParent(note) == parent {
			files = append(files, note)
		}
	}
	dirNames := make([]string, 0, len(dirSet))
	for name := range dirSet {
		dirNames = append(dirNames, name)
	}
	sort.Slice(dirNames, func(i, j int) bool {
		return strings.ToLower(dirNames[i]) < strings.ToLower(dirNames[j])
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i]) < strings.ToLower(files[j])
	})

	var entries []treeEntry
	for _, name := range dirNames {
		path := dirSet[name]
		id := "vault:" + path
		entries = append(entries, treeEntry{
			name:  name + "/",
			path:  path,
			id:    id,
			isDir: true,
			vault: true,
			depth: depth,
		})
		if expanded[id] {
			entries = append(entries, vaultTreeChildren(path, depth+1, notes, folders, expanded)...)
		}
	}
	for _, note := range files {
		entries = append(entries, treeEntry{
			name:  vaultBase(note),
			path:  note,
			id:    "vault:" + note,
			vault: true,
			depth: depth,
		})
	}
	return entries
}

func addVaultParents(path string, folders map[string]bool) {
	parent := vaultParent(path)
	for parent != "" {
		folders[parent] = true
		parent = vaultParent(parent)
	}
}

func vaultParent(path string) string {
	path = cleanVaultPath(path)
	if path == "" || !strings.Contains(path, "/") {
		return ""
	}
	return path[:strings.LastIndex(path, "/")]
}

func vaultBase(path string) string {
	path = strings.TrimSuffix(cleanVaultPath(path), "/")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func cleanVaultPath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimPrefix(path, "/")
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}

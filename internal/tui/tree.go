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

func readTree(dir string, vaultNotes []string, expanded map[string]bool) ([]treeEntry, error) {
	var entries []treeEntry
	entries = append(entries, treeEntry{name: "Vault", id: "vault:", isDir: true, vault: true})
	if expanded["vault:"] {
		entries = append(entries, vaultTreeEntries(vaultNotes, expanded)...)
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
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".md" || ext == ".markdown" || ext == ".txt" {
			entries = append(entries, treeEntry{name: name, path: path, id: "file:" + path, depth: depth})
		}
	}
	return entries, nil
}

func vaultTreeEntries(notes []string, expanded map[string]bool) []treeEntry {
	sort.Slice(notes, func(i, j int) bool {
		return strings.ToLower(notes[i]) < strings.ToLower(notes[j])
	})
	seenDirs := map[string]bool{}
	hiddenDir := map[string]bool{}
	var entries []treeEntry
	for _, note := range notes {
		clean := cleanVaultPath(note)
		if clean == "" {
			continue
		}
		parts := strings.Split(clean, "/")
		var prefix []string
		hidden := false
		for i := 0; i < len(parts)-1; i++ {
			prefix = append(prefix, parts[i])
			path := strings.Join(prefix, "/")
			if hidden || hiddenDir[path] {
				hidden = true
				continue
			}
			if seenDirs[path] {
				if !expanded["vault:"+path] {
					hidden = true
				}
				continue
			}
			seenDirs[path] = true
			entries = append(entries, treeEntry{
				name:  parts[i] + "/",
				path:  path + "/",
				id:    "vault:" + path,
				isDir: true,
				vault: true,
				depth: i + 1,
			})
			if !expanded["vault:"+path] {
				hidden = true
				hiddenDir[path] = true
			}
		}
		if hidden {
			continue
		}
		entries = append(entries, treeEntry{
			name:  parts[len(parts)-1],
			path:  clean,
			id:    "vault:" + clean,
			vault: true,
			depth: len(parts),
		})
	}
	return entries
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

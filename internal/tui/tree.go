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
	isDir bool
	vault bool
	depth int
}

func readTree(dir string, vaultRoot string, vaultNotes []string) ([]treeEntry, error) {
	var entries []treeEntry
	entries = append(entries, treeEntry{name: "Vault", path: vaultRoot, isDir: true, vault: true})
	entries = append(entries, vaultTreeEntries(vaultNotes)...)
	entries = append(entries, treeEntry{name: ".", path: dir, isDir: true})

	items, err := os.ReadDir(dir)
	if err != nil {
		return entries, err
	}
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
			entries = append(entries, treeEntry{name: name + "/", path: path, isDir: true})
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".md" || ext == ".markdown" || ext == ".txt" {
			entries = append(entries, treeEntry{name: name, path: path})
		}
	}
	return entries, nil
}

func vaultTreeEntries(notes []string) []treeEntry {
	sort.Slice(notes, func(i, j int) bool {
		return strings.ToLower(notes[i]) < strings.ToLower(notes[j])
	})
	seenDirs := map[string]bool{}
	var entries []treeEntry
	for _, note := range notes {
		clean := cleanVaultPath(note)
		if clean == "" {
			continue
		}
		parts := strings.Split(clean, "/")
		var prefix []string
		for i := 0; i < len(parts)-1; i++ {
			prefix = append(prefix, parts[i])
			path := strings.Join(prefix, "/")
			if seenDirs[path] {
				continue
			}
			seenDirs[path] = true
			entries = append(entries, treeEntry{
				name:  parts[i] + "/",
				path:  path + "/",
				isDir: true,
				vault: true,
				depth: i + 1,
			})
		}
		entries = append(entries, treeEntry{
			name:  parts[len(parts)-1],
			path:  clean,
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

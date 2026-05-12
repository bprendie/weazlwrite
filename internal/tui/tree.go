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
}

func readTree(dir string, vaultRoot string, vaultNotes []string) ([]treeEntry, error) {
	var entries []treeEntry
	entries = append(entries, treeEntry{name: "Vault", path: vaultRoot, isDir: true, vault: true})
	for _, note := range vaultNotes {
		entries = append(entries, treeEntry{name: "  " + note, path: note, vault: true})
	}
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

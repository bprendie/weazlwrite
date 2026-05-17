package storage

import (
	"path/filepath"
	"strings"
)

func renamedChildPath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	return newPath + strings.TrimPrefix(path, oldPath)
}

func cleanStorePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimPrefix(path, "/")
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}

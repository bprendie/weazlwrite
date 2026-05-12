package tui

import (
	"path/filepath"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	return wordwrap.String(s, width)
}

func titleFor(path string, content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		if line != "" {
			return minString(line, 60)
		}
	}
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "Untitled"
	}
	return base
}

func minString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

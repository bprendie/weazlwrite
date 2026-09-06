package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"strings"

	"github.com/rivo/uniseg"
)

// Editor layout and hit testing share the buffer's grapheme-aware wrap.
func wrapRunes(runes []rune, width int) [][]rune { return textarea.WrapRunes(runes, width) }

func visualRowCount(value string, width int) int {
	n := 0
	for _, line := range strings.Split(value, "\n") {
		n += len(wrapRunes([]rune(line), width))
	}
	if n < 1 {
		return 1
	}
	return n
}

func cursorVisualRow(value string, logicalLine, rowOffset, width int) int {
	row := 0
	lines := strings.Split(value, "\n")
	limit := min(max(0, logicalLine), max(0, len(lines)-1))
	for i := 0; i < limit; i++ {
		row += len(wrapRunes([]rune(lines[i]), width))
	}
	return row + max(0, rowOffset)
}

func logicalAtVisualRow(value string, target, width int) (line, offset int) {
	if target < 0 {
		target = 0
	}
	lines := strings.Split(value, "\n")
	vis := 0
	for i, ln := range lines {
		h := len(wrapRunes([]rune(ln), width))
		if vis+h > target {
			return i, target - vis
		}
		vis += h
		line = i
		offset = max(0, h-1)
	}
	return line, offset
}

func runeIndexAtVisual(line string, rowOff, colX, width int) int {
	runes := []rune(line)
	rows := wrapRunes(runes, width)
	if len(rows) == 0 {
		return 0
	}
	rowOff = min(max(0, rowOff), len(rows)-1)
	idx := 0
	for r := 0; r < rowOff; r++ {
		idx += len(rows[r])
	}
	x := 0
	g := uniseg.NewGraphemes(string(rows[rowOff]))
	for g.Next() {
		if x+g.Width() > colX {
			break
		}
		x += g.Width()
		idx += len([]rune(g.Str()))
	}
	return min(idx, len(runes))
}

package tui

import (
	"strings"
	"unicode"

	rw "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// wrapRunes word-wraps a logical line the way bubbles textarea does, including
// the trailing space it keeps on each soft-wrapped row.
func wrapRunes(runes []rune, width int) [][]rune {
	if width < 1 {
		width = 1
	}
	var (
		lines  = [][]rune{{}}
		word   = []rune{}
		row    int
		spaces int
	)
	for _, r := range runes {
		if unicode.IsSpace(r) {
			spaces++
		} else {
			word = append(word, r)
		}
		if spaces > 0 {
			if uniseg.StringWidth(string(lines[row]))+uniseg.StringWidth(string(word))+spaces > width {
				row++
				lines = append(lines, []rune{})
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatRunes(' ', spaces)...)
				spaces = 0
				word = nil
			} else {
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatRunes(' ', spaces)...)
				spaces = 0
				word = nil
			}
			continue
		}
		lastCharLen := rw.RuneWidth(word[len(word)-1])
		if uniseg.StringWidth(string(word))+lastCharLen > width {
			if len(lines[row]) > 0 {
				row++
				lines = append(lines, []rune{})
			}
			lines[row] = append(lines[row], word...)
			word = nil
		}
	}
	if uniseg.StringWidth(string(lines[row]))+uniseg.StringWidth(string(word))+spaces >= width {
		lines = append(lines, []rune{})
		lines[row+1] = append(lines[row+1], word...)
		spaces++
		lines[row+1] = append(lines[row+1], repeatRunes(' ', spaces)...)
	} else {
		lines[row] = append(lines[row], word...)
		spaces++
		lines[row] = append(lines[row], repeatRunes(' ', spaces)...)
	}
	return lines
}

func repeatRunes(r rune, n int) []rune {
	if n <= 0 {
		return nil
	}
	return []rune(strings.Repeat(string(r), n))
}

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
		n := len([]rune(strings.TrimRight(string(rows[r]), " ")))
		idx += n
		if idx < len(runes) && unicode.IsSpace(runes[idx]) {
			idx++
		}
	}
	x := 0
	for _, r := range []rune(strings.TrimRight(string(rows[rowOff]), " ")) {
		w := rw.RuneWidth(r)
		if x >= colX {
			break
		}
		if x+w > colX && x > 0 {
			break
		}
		x += w
		idx++
	}
	return min(max(0, idx), len(runes))
}

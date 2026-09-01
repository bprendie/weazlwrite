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
	if value == "" {
		return 1
	}
	n := 0
	for _, line := range strings.Split(value, "\n") {
		n += len(wrapRunes([]rune(line), width))
	}
	return n
}

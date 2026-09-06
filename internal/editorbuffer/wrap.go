package textarea

import (
	"github.com/rivo/uniseg"
	"strings"
)

func (m Model) memoizedWrap(runes []rune, width int) [][]rune {
	input := line{runes: runes, width: width}
	if v, ok := m.cache.Get(input); ok {
		return v
	}
	v := wrap(runes, width)
	m.cache.Set(input, v)
	return v
}

// cursorLineNumber returns the line number that the cursor is on.
// This accounts for soft wrapped lines.
func (m Model) cursorLineNumber() int {
	starts, _ := m.RowStarts()
	return starts[m.row] + m.LineInfo().RowOffset
}

// WrapRunes preserves every source rune and keeps grapheme clusters intact.
// The final virtual space provides a cursor cell at the end of a paragraph.
func WrapRunes(runes []rune, width int) [][]rune {
	width = max(1, width)
	rows := [][]rune{{}}
	cells := 0
	appendChunk := func(text string) {
		g := uniseg.NewGraphemes(text)
		for g.Next() {
			if cells+g.Width() > width && cells > 0 {
				rows = append(rows, []rune{})
				cells = 0
			}
			rows[len(rows)-1] = append(rows[len(rows)-1], []rune(g.Str())...)
			cells += g.Width()
		}
	}
	var word strings.Builder
	flush := func() {
		if cells > 0 && cells+uniseg.StringWidth(word.String()) > width {
			rows = append(rows, []rune{})
			cells = 0
		}
		appendChunk(word.String())
		word.Reset()
	}
	g := uniseg.NewGraphemes(string(runes))
	for g.Next() {
		if strings.TrimSpace(g.Str()) == "" {
			flush()
			appendChunk(g.Str())
		} else {
			word.WriteString(g.Str())
		}
	}
	if word.Len() != 0 {
		flush()
	}
	appendChunk(" ")
	return rows
}

func wrap(runes []rune, width int) [][]rune { return WrapRunes(runes, width) }

func repeatSpaces(n int) []rune {
	return []rune(strings.Repeat(string(' '), n))
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

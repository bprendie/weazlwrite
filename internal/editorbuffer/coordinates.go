package textarea

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

type Position struct{ Line, Column int }
type Highlight struct {
	Start, End Position
	Style      lipgloss.Style
}

func (m *Model) SetHighlights(ranges []Highlight) { m.highlights = ranges }
func (m Model) CursorPosition() Position          { return Position{m.row, m.col} }
func (m Model) YOffset() int                      { return m.viewport.YOffset }
func (m Model) VisualRow() int                    { return m.cursorLineNumber() }

func (m *Model) SetPosition(p Position) {
	m.row = clamp(p.Line, 0, len(m.value)-1)
	m.SetCursor(clamp(p.Column, 0, len(m.value[m.row])))
	m.repositionView()
}

func (m Model) VisualRows() int { _, total := m.RowStarts(); return total }

func (m *Model) ScrollRows(delta int) {
	m.viewport.YOffset = clamp(m.viewport.YOffset+delta, 0, max(0, m.VisualRows()-m.height))
}

func (m Model) PositionAt(visualRow, cell int) Position {
	visualRow, cell = max(0, visualRow), max(0, cell)
	for l, line := range m.value {
		start := 0
		for _, row := range m.memoizedWrap(line, m.width) {
			if visualRow == 0 {
				col, x := start, 0
				g := uniseg.NewGraphemes(string(row))
				for g.Next() {
					if x+g.Width() > cell {
						break
					}
					x += g.Width()
					col += utf8.RuneCountInString(g.Str())
				}
				return Position{l, min(col, len(line))}
			}
			visualRow--
			start += len(row)
		}
	}
	last := len(m.value) - 1
	return Position{last, len(m.value[last])}
}

func (m Model) AdjacentPosition(direction int) Position {
	p := m.CursorPosition()
	if direction < 0 {
		if p.Column == 0 {
			if p.Line > 0 {
				return Position{p.Line - 1, len(m.value[p.Line-1])}
			}
			return p
		}
		previous, at := 0, 0
		g := uniseg.NewGraphemes(string(m.value[p.Line]))
		for g.Next() {
			at += utf8.RuneCountInString(g.Str())
			if at >= p.Column {
				return Position{p.Line, previous}
			}
			previous = at
		}
	} else {
		if p.Column == len(m.value[p.Line]) {
			if p.Line+1 < len(m.value) {
				return Position{p.Line + 1, 0}
			}
			return p
		}
		at := 0
		g := uniseg.NewGraphemes(string(m.value[p.Line]))
		for g.Next() {
			at += utf8.RuneCountInString(g.Str())
			if at > p.Column {
				return Position{p.Line, at}
			}
		}
	}
	return p
}

func before(a, b Position) bool { return a.Line < b.Line || a.Line == b.Line && a.Column < b.Column }

func (m Model) renderWrappedLine(base lipgloss.Style, row []rune, line, start int) string {
	var out strings.Builder
	g := uniseg.NewGraphemes(string(row))
	at := start
	for g.Next() {
		length := utf8.RuneCountInString(g.Str())
		style := base
		for _, h := range m.highlights {
			if before(Position{line, at}, h.End) && before(h.Start, Position{line, at + length}) {
				style = h.Style.Inherit(base)
			}
		}
		if m.Focused() && m.row == line && m.col >= at && m.col < at+length {
			m.Cursor.TextStyle = style
			m.Cursor.SetChar(g.Str())
			out.WriteString(style.Render(m.Cursor.View()))
		} else {
			out.WriteString(style.Render(g.Str()))
		}
		at += length
	}
	return out.String()
}

func (m *Model) snapCursor() {
	at := 0
	g := uniseg.NewGraphemes(string(m.value[m.row]))
	for g.Next() {
		end := at + utf8.RuneCountInString(g.Str())
		if m.col > at && m.col < end {
			m.col = at
			return
		}
		at = end
	}
}

type rowLayout struct {
	version uint64
	width   int
	starts  []int
	total   int
}

func (m Model) RowStarts() ([]int, int) {
	if m.layout != nil && m.layout.version == m.layoutVersion && m.layout.width == m.width && len(m.layout.starts) == len(m.value) {
		return m.layout.starts, m.layout.total
	}
	starts := make([]int, len(m.value))
	total := 0
	for i, line := range m.value {
		starts[i] = total
		total += len(m.memoizedWrap(line, m.width))
	}
	if m.layout != nil {
		*m.layout = rowLayout{version: m.layoutVersion, width: m.width, starts: starts, total: total}
	}
	return starts, total
}

func (m *Model) SetYOffset(offset int) {
	m.viewport.YOffset = clamp(offset, 0, max(0, m.VisualRows()-1))
}

package textarea

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"strconv"
	"strings"
)

// View renders the text area in its current state.
func (m Model) View() string {
	if m.Value() == "" && m.row == 0 && m.col == 0 && m.Placeholder != "" {
		return m.placeholderView()
	}
	starts, total := m.RowStarts()
	if m.viewport.TotalLineCount() != total+m.height+1 {
		m.viewport.SetContent(strings.Repeat("\n", total+m.height))
	}
	offset := m.viewport.YOffset
	var out []string
	for l, line := range m.value {
		end := total
		if l+1 < len(starts) {
			end = starts[l+1]
		}
		if end <= offset {
			continue
		}
		style := m.style.computedText()
		if l == m.row {
			style = m.style.computedCursorLine()
		}
		start := 0
		for r, row := range m.memoizedWrap(line, m.width) {
			visual := starts[l] + r
			if visual >= offset && len(out) < m.height {
				prefix := m.style.computedPrompt().Render(m.getPromptString(visual))
				if m.ShowLineNumbers {
					n := any(" ")
					if r == 0 {
						n = l + 1
					}
					prefix += m.style.computedLineNumber().Render(m.formatLineNumber(n))
				}
				text := m.renderWrappedLine(style, row, l, start)
				text = ansi.Truncate(text, m.width, "")
				out = append(out, prefix+text+style.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(text)))))
			}
			start += len(row)
		}
		if len(out) >= m.height {
			break
		}
	}
	for len(out) < m.height {
		out = append(out, m.style.computedPrompt().Render(m.getPromptString(offset+len(out)))+strings.Repeat(" ", m.width))
	}
	return m.style.Base.Render(strings.Join(out, "\n"))
}

// formatLineNumber formats the line number for display dynamically based on
// the maximum number of lines.
func (m Model) formatLineNumber(x any) string {
	// XXX: ultimately we should use a max buffer height, which has yet to be
	// implemented.
	digits := len(strconv.Itoa(m.MaxHeight))
	return fmt.Sprintf(" %*v ", digits, x)
}

func (m Model) getPromptString(displayLine int) (prompt string) {
	prompt = m.Prompt
	if m.promptFunc == nil {
		return prompt
	}
	prompt = m.promptFunc(displayLine)
	pl := uniseg.StringWidth(prompt)
	if pl < m.promptWidth {
		prompt = fmt.Sprintf("%*s%s", m.promptWidth-pl, "", prompt)
	}
	return prompt
}

// placeholderView returns the prompt and placeholder view, if any.
func (m Model) placeholderView() string {
	var (
		s     strings.Builder
		p     = m.Placeholder
		style = m.style.computedPlaceholder()
	)

	// word wrap lines
	pwordwrap := ansi.Wordwrap(p, m.width, "")
	// wrap lines (handles lines that could not be word wrapped)
	pwrap := ansi.Hardwrap(pwordwrap, m.width, true)
	// split string by new lines
	plines := strings.Split(strings.TrimSpace(pwrap), "\n")

	for i := 0; i < m.height; i++ {
		lineStyle := m.style.computedPlaceholder()
		lineNumberStyle := m.style.computedLineNumber()
		if len(plines) > i {
			lineStyle = m.style.computedCursorLine()
			lineNumberStyle = m.style.computedCursorLineNumber()
		}

		// render prompt
		prompt := m.getPromptString(i)
		prompt = m.style.computedPrompt().Render(prompt)
		s.WriteString(lineStyle.Render(prompt))

		// when show line numbers enabled:
		// - render line number for only the cursor line
		// - indent other placeholder lines
		// this is consistent with vim with line numbers enabled
		if m.ShowLineNumbers {
			var ln string

			switch {
			case i == 0:
				ln = strconv.Itoa(i + 1)
				fallthrough
			case len(plines) > i:
				s.WriteString(lineStyle.Render(lineNumberStyle.Render(m.formatLineNumber(ln))))
			default:
			}
		}

		switch {
		// first line
		case i == 0:
			// first character of first line as cursor with character
			m.Cursor.TextStyle = m.style.computedPlaceholder()

			ch, rest, _, _ := uniseg.FirstGraphemeClusterInString(plines[0], 0)
			m.Cursor.SetChar(ch)
			s.WriteString(lineStyle.Render(m.Cursor.View()))

			// the rest of the first line
			s.WriteString(lineStyle.Render(style.Render(rest)))
		// remaining lines
		case len(plines) > i:
			// current line placeholder text
			if len(plines) > i {
				s.WriteString(lineStyle.Render(style.Render(plines[i] + strings.Repeat(" ", max(0, m.width-uniseg.StringWidth(plines[i]))))))
			}
		default:
			// end of line buffer character
			eob := m.style.computedEndOfBuffer().Render(string(m.EndOfBufferCharacter))
			s.WriteString(eob)
		}

		// terminate with new line
		s.WriteRune('\n')
	}

	m.viewport.SetContent(s.String())
	return m.style.Base.Render(m.viewport.View())
}

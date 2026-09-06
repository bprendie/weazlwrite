package tui

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var listPrefix = regexp.MustCompile(`^([ \t]*)((?:>[ \t]*)*)([-+*]|[0-9]{1,9}[.)])[ \t]+(\[[ xX]\][ \t]+)?`)
var quotePrefix = regexp.MustCompile(`^([ \t]*)((?:>[ \t]*)+)`)

type markdownPrefix struct {
	indent, quote, marker, task, full string
}

func parseMarkdownPrefix(line string) markdownPrefix {
	if parts := listPrefix.FindStringSubmatch(line); parts != nil {
		return markdownPrefix{parts[1], parts[2], parts[3], parts[4], parts[0]}
	}
	if parts := quotePrefix.FindStringSubmatch(line); parts != nil {
		return markdownPrefix{indent: parts[1], quote: parts[2], full: parts[0]}
	}
	return markdownPrefix{indent: leadingIndent(line)}
}

func leadingIndent(line string) string { return line[:len(line)-len(strings.TrimLeft(line, " \t"))] }

// Check the opening fence as well as its body; Enter on a closing fence
// resumes ordinary prose behavior. Longer fences may contain shorter ones.
func inCodeFence(lines []string, at int) bool {
	var fence byte
	size := 0
	for i := 0; i <= at && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if prefix := quotePrefix.FindString(line); prefix != "" {
			line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
		if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
			continue
		}
		n := 0
		for n < len(line) && line[n] == line[0] {
			n++
		}
		if n < 3 {
			continue
		}
		if fence == 0 {
			fence, size = line[0], n
		} else if line[0] == fence && n >= size && strings.TrimSpace(line[n:]) == "" {
			fence, size = 0, 0
		}
	}
	return fence != 0
}

func (m *model) replaceEditorLine(line int, text string, col int) {
	lines := strings.Split(m.editorText(), "\n")
	lines[line] = text
	m.setEditorText(strings.Join(lines, "\n"))
	m.moveEditorToLine(line)
	m.editor.SetCursor(col)
}

func (m *model) markdownEnter() {
	lines := strings.Split(m.editorText(), "\n")
	row, col := m.editor.Line(), m.editorCursorCol()
	line := lines[row]
	p := parseMarkdownPrefix(line)
	if inCodeFence(lines, row) {
		m.editor.InsertString(hideTabs("\n" + leadingIndent(line) + p.quote))
		return
	}
	if p.full == "" || col < utf8.RuneCountInString(p.full) {
		m.editor.InsertString("\n")
		return
	}
	if strings.TrimSpace(strings.TrimPrefix(line, p.full)) == "" {
		if p.indent != "" {
			m.outdentEditorLine()
			return
		}
		replacement := p.quote
		if p.marker == "" {
			last := strings.LastIndex(replacement, ">")
			replacement = replacement[:last]
		}
		m.replaceEditorLine(row, replacement, utf8.RuneCountInString(replacement))
		return
	}
	marker := p.marker
	if len(marker) > 1 {
		n, _ := strconv.Atoi(marker[:len(marker)-1])
		marker = strconv.Itoa(n+1) + marker[len(marker)-1:]
	}
	if marker != "" {
		marker += " "
	}
	task := ""
	if p.task != "" {
		task = "[ ] "
	}
	m.editor.InsertString(hideTabs("\n" + p.indent + p.quote + marker + task))
}

func (m *model) markdownTab() {
	lines := strings.Split(m.editorText(), "\n")
	row, col := m.editor.Line(), m.editorCursorCol()
	if !inCodeFence(lines, row) && parseMarkdownPrefix(lines[row]).marker != "" {
		m.replaceEditorLine(row, strings.Repeat(" ", editorIndentWidth)+lines[row], col+editorIndentWidth)
		return
	}
	m.editor.InsertString(strings.Repeat(" ", editorIndentWidth))
}

func (m *model) markdownBackspace() bool {
	lines := strings.Split(m.editorText(), "\n")
	row := m.editor.Line()
	p := parseMarkdownPrefix(lines[row])
	if inCodeFence(lines, row) || p.full == "" || m.editorCursorCol() != utf8.RuneCountInString(p.full) {
		return false
	}
	if p.indent != "" {
		m.outdentEditorLine()
		return true
	}
	replacement := p.quote
	if p.marker == "" {
		replacement = replacement[:strings.LastIndex(replacement, ">")]
	}
	m.replaceEditorLine(row, replacement+strings.TrimPrefix(lines[row], p.full), utf8.RuneCountInString(replacement))
	return true
}

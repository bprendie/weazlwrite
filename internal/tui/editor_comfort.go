package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"

	"github.com/bprendie/weazlwrite/internal/config"
)

func (m model) editorLayout() (width, padding int) {
	_, mainW := m.layoutWidths()
	available := contentWidth(m.mainPanelStyle(), mainW)
	width = available
	if desired := m.cfg.UI.EditorWidth; desired > 0 {
		width = min(available, max(20, desired)+editorGutterWidth)
	}
	return width, max(0, (available-width)/2)
}

func (m *model) cycleWritingWidth() {
	switch m.cfg.UI.EditorWidth {
	case 0:
		m.cfg.UI.EditorWidth = 80
	case 80:
		m.cfg.UI.EditorWidth = 90
	case 90:
		m.cfg.UI.EditorWidth = 100
	default:
		m.cfg.UI.EditorWidth = 0
	}
	m.resize()
	m.status = fmt.Sprintf("writing width: %d columns", m.cfg.UI.EditorWidth)
	if m.cfg.UI.EditorWidth == 0 {
		m.status = "writing width: full pane"
	}
	m.saveWritingPreferences()
}

func (m *model) toggleTypewriter() {
	m.cfg.UI.EditorTypewriter = !m.cfg.UI.EditorTypewriter
	m.editor.Typewriter = m.cfg.UI.EditorTypewriter
	m.editor.SetPosition(m.editor.CursorPosition())
	m.status = "typewriter scrolling off"
	if m.cfg.UI.EditorTypewriter {
		m.status = "typewriter scrolling on"
	}
	m.saveWritingPreferences()
}

func (m *model) saveWritingPreferences() {
	if m.cfgPath != "" {
		if err := config.Save(m.cfgPath, m.cfg); err != nil {
			m.err = "writing preferences: " + err.Error()
		}
	}
}

func (m *model) alignPreviewToEditor() {
	lines := strings.Split(m.editorText(), "\n")
	source := strings.TrimSpace(lines[min(m.editor.Line(), len(lines)-1)])
	source = strings.TrimLeft(source, "# >-*_`")
	needle := []rune(source)
	if len(needle) > 24 {
		needle = needle[:24]
	}
	rendered := strings.Split(m.markdown.Render(m.editorText(), m.preview.Width), "\n")
	expected := m.editor.VisualRow() * len(rendered) / max(1, m.editor.VisualRows())
	best, distance := -1, len(rendered)+1
	if len(needle) > 3 {
		for i, line := range rendered {
			if strings.Contains(ansi.Strip(line), string(needle)) {
				d := i - expected
				if d < 0 {
					d = -d
				}
				if d < distance {
					best, distance = i, d
				}
			}
		}
	}
	if best >= 0 {
		m.preview.SetYOffset(max(0, best-2))
	} else {
		m.preview.SetYOffset(expected)
	}
}

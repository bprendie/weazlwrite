package tui

import (
	"strings"
	"testing"

	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func searchModel() model {
	m := inputModel()
	m.findPrompt = textinput.New()
	m.width, m.height = 100, 30
	m.resize()
	return m
}

func TestInlineFindReplaceAndUndo(t *testing.T) {
	m := searchModel()
	m.loadEditorText("alpha beta alpha")
	m.editor.SetCursor(0)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	m, _ = updateModel(m, textKey("alpha"))
	if m.mode != modeFind || len(m.search.matches) != 2 || !strings.Contains(ansi.Strip(m.View()), "alpha beta alpha") {
		t.Fatal("inline search hid the document or missed matches")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.editorCursorCol() != 11 {
		t.Fatal("Enter did not advance")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyF15})
	if m.editorCursorCol() != 0 {
		t.Fatal("previous did not return")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = updateModel(m, textKey("gamma"))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.editorText() != "gamma beta alpha" || !m.dirty {
		t.Fatalf("replace: %q", m.editorText())
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	m.undoEdit()
	if m.editorText() != "alpha beta alpha" {
		t.Fatal("replace undo failed")
	}
}

func TestSearchCancelRestoresPositionAndAcceptSelectsMatch(t *testing.T) {
	m := searchModel()
	m.loadEditorText(strings.Repeat("needle haystack\n", 60))
	m.editor.SetPosition(textarea.Position{Line: 21, Column: 3})
	m.editor.SetYOffset(18)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	m, _ = updateModel(m, textKey("needle"))
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.editor.Line() != 21 || m.editorCursorCol() != 3 || m.editor.YOffset() != 18 {
		t.Fatal("cancel lost writing position")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyF2})
	if !m.hasTextSelection() || m.textBetween(m.dragStart, m.dragEnd) != "needle" {
		t.Fatal("F2 did not select match for editing")
	}
	m, _ = updateModel(m, textKey("thread"))
	if !strings.Contains(m.editorText(), "thread haystack") {
		t.Fatal("accepted match could not be rewritten")
	}
}

func TestSearchWhitespaceUnicodeAndRepeat(t *testing.T) {
	m := searchModel()
	m.loadEditorText("日本 foo 日本 foo")
	m.editor.SetCursor(0)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	m, _ = updateModel(m, textKey("日本"))
	if len(m.search.matches) != 2 {
		t.Fatal("Unicode matches wrong")
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyF2})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyF3})
	a, _ := m.selectionBounds()
	if a.col != 7 {
		t.Fatalf("next Unicode match column %d", a.col)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyF15})
	a, _ = m.selectionBounds()
	if a.col != 0 {
		t.Fatal("previous match failed")
	}
	m.lastFind = " "
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
	if len(m.search.matches) != 3 {
		t.Fatal("whitespace query was trimmed")
	}
}

func TestWritingWidthKeepsMouseCoordinatesAligned(t *testing.T) {
	m := searchModel()
	m.width = 180
	m.cfg.UI.EditorWidth = 80
	m.resize()
	m.loadEditorText("click here")
	width, pad := m.editorLayout()
	if width != 80+editorGutterWidth || pad == 0 {
		t.Fatal("prose width not centered")
	}
	cx, cy, _, _ := m.mainContentBounds()
	p, ok := m.editorPosAt(cx+editorGutterWidth+6, cy)
	if !ok || p.col != 6 {
		t.Fatalf("padded click: %+v", p)
	}
	m.width = 35
	m.resize()
	width, pad = m.editorLayout()
	if width > 35 || pad != 0 {
		t.Fatal("narrow terminal overflowed")
	}
}

func TestInlineSearchFitsNarrowAndWidePanes(t *testing.T) {
	for _, width := range []int{40, 80, 120, 180} {
		m := searchModel()
		m.width = width
		m.cfg.UI.EditorWidth = 90
		m.resize()
		m.loadEditorText("alpha beta alpha")
		m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlF})
		m, _ = updateModel(m, textKey("alpha"))
		m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
		m, _ = updateModel(m, textKey(strings.Repeat("replacement ", 20)))
		_, mainWidth := m.layoutWidths()
		bar := m.editorSearchView(contentWidth(m.mainPanelStyle(), mainWidth))
		if len(strings.Split(bar, "\n")) != 2 {
			t.Fatal("search bar did not occupy two rows")
		}
		for _, line := range strings.Split(bar, "\n") {
			if lipgloss.Width(line) > contentWidth(m.mainPanelStyle(), mainWidth) {
				t.Fatal("search bar overflows pane")
			}
		}
		if lipgloss.Height(m.View()) > m.height {
			t.Fatal("search overflows terminal height")
		}
	}
}

func TestWriterPanelsMatchHitTestBounds(t *testing.T) {
	m := searchModel()
	m.width = 120
	m.treeVisible = true
	m.resize()
	m.loadEditorText("alpha")
	if got := lipgloss.Width(m.writeView()); got != m.width {
		t.Fatalf("panels occupy %d columns, hit testing expects %d", got, m.width)
	}
}

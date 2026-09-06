package textarea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestWrapPreservesSourceAndGraphemes(t *testing.T) {
	source := "日本e\u0301👩‍💻  long-word"
	for _, width := range []int{2, 4, 10} {
		rows := WrapRunes([]rune(source), width)
		var joined string
		for _, row := range rows {
			joined += string(row)
		}
		if joined != source+" " {
			t.Fatalf("wrap lost whitespace or content: %q", joined)
		}
		found := false
		for _, row := range rows {
			if strings.Contains(string(row), "👩‍💻") {
				found = true
			}
		}
		if !found {
			t.Fatal("wrap split a joined emoji")
		}
	}
}

func TestScrollMarginTypewriterAndWheelPosition(t *testing.T) {
	m := New()
	m.ShowLineNumbers = false
	m.SetWidth(30)
	m.SetHeight(8)
	m.Focus()
	m.SetValue(strings.Repeat("line\n", 40))
	m.ScrollMargin = 3
	m.SetPosition(Position{20, 0})
	if m.VisualRow()-m.YOffset() != 3 && m.VisualRow()-m.YOffset() != 4 {
		t.Fatal("margin did not leave room around caret")
	}
	m.Typewriter = true
	m.SetPosition(Position{25, 0})
	if m.YOffset() != 21 {
		t.Fatalf("typewriter offset=%d", m.YOffset())
	}
	m.ScrollRows(-3)
	offset := m.YOffset()
	m, _ = m.Update(struct{}{})
	if m.YOffset() != offset {
		t.Fatal("non-key event undid mouse scroll")
	}
}

func TestLayoutCacheInvalidatesOnEditAndResize(t *testing.T) {
	m := New()
	m.ShowLineNumbers = false
	m.SetWidth(20)
	m.Focus()
	m.SetValue("short")
	initial := m.VisualRows()
	m.InsertString(strings.Repeat(" word", 20))
	if m.VisualRows() <= initial {
		t.Fatal("insertion left stale layout")
	}
	before := m.VisualRows()
	m.SetWidth(10)
	if m.VisualRows() <= before {
		t.Fatal("resize left stale layout")
	}
	m.SetValue("new")
	if m.VisualRows() != 1 {
		t.Fatal("load left stale layout")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.VisualRows() != 2 {
		t.Fatal("Enter left stale layout")
	}
}

func TestVisibleRowsRenderAfterScrolling(t *testing.T) {
	m := New()
	m.ShowLineNumbers = false
	m.Prompt = ""
	m.SetWidth(20)
	m.SetHeight(3)
	m.SetValue("zero\none\ntwo\nthree\nfour\nfive")
	m.SetYOffset(2)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "two") || !strings.Contains(view, "four") || strings.Contains(view, "zero") || strings.Contains(view, "five") {
		t.Fatalf("wrong visible rows: %q", view)
	}
}

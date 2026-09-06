package tui

import (
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/charmbracelet/bubbles/viewport"
	"strings"
	"testing"
)

func TestSelectionBoundsExcludeTreeAndSelectedTextExcludesLineNumbers(t *testing.T) {
	editor := textarea.New()
	editor.SetValue("alpha\nbeta\ngamma")
	m := model{
		styles:      newStyles(),
		mode:        modeWrite,
		focus:       focusEditor,
		view:        viewEdit,
		treeVisible: true,
		width:       100,
		height:      24,
		editor:      editor,
		preview:     viewport.New(0, 0),
	}
	m.resize()
	contentX, contentY, _, _ := m.mainContentBounds()
	if _, ok := m.selectionRowAt(1, contentY); ok {
		t.Fatal("selection row accepted x inside tree")
	}
	row, ok := m.selectionRowAt(contentX, contentY+1)
	if !ok {
		t.Fatal("selection row rejected x inside writing pane")
	}
	if row != 1 {
		t.Fatalf("selection row = %d, want 1", row)
	}
	m.selectStart = selectPoint{row: 0}
	m.selectEnd = selectPoint{row: 1}
	if got := m.selectedText(); got != "alpha\nbeta" {
		t.Fatalf("selectedText = %q, want editor content without line numbers", got)
	}
}

func TestSelectionViewWrapsLongLine(t *testing.T) {
	editor := textarea.New()
	long := strings.Repeat("word ", 30)
	editor.SetValue(long)
	m := model{
		styles: newStyles(),
		mode:   modeWrite,
		focus:  focusEditor,
		view:   viewEdit,
		editor: editor,
	}
	m.selectStart = selectPoint{row: 0}
	m.selectEnd = selectPoint{row: 0}
	got := m.selectionView(20, 12)
	if strings.Count(got, "\n") < 1 {
		t.Fatalf("selection view did not wrap, got %q", got)
	}
	if !strings.Contains(got, "word") {
		t.Fatalf("selection view missing content: %q", got)
	}
	if got := m.selectedText(); got != long {
		t.Fatalf("selectedText copied wrapped visual rows")
	}
}

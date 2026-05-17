package tui

func (m *model) renderPreview() {
	content := m.editor.Value()
	width := max(10, m.preview.Width)
	m.preview.SetContent(m.markdown.Render(content, width))
}

func (m *model) resize() {
	innerH := m.bodyHeight()
	_, mainW := m.layoutWidths()
	m.editor.ShowLineNumbers = !m.selectionMode
	m.editor.SetWidth(contentWidth(m.styles.panel, mainW))
	m.editor.SetHeight(contentHeight(m.styles.panel, innerH))
	m.preview.Width = contentWidth(m.styles.panel, mainW)
	m.preview.Height = contentHeight(m.styles.panel, innerH)
	m.helpView.Width = contentWidth(m.styles.panel, m.width)
	m.helpView.Height = contentHeight(m.styles.panel, innerH)
	m.markdown.Resize(m.preview.Width)
	if m.view == viewRender {
		m.renderPreview()
	}
}

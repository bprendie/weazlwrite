package tui

func (m *model) renderPreview() {
	content := m.editorText()
	width := max(10, m.preview.Width)
	m.preview.SetContent(m.markdown.Render(content, width))
}

func (m *model) resize() {
	innerH := m.bodyHeight()
	_, mainW := m.layoutWidths()
	m.configureEditor()
	mainStyle := m.mainPanelStyle()
	editorWidth, _ := m.editorLayout()
	m.editor.SetWidth(editorWidth)
	editorHeight := contentHeight(mainStyle, innerH)
	if m.mode == modeFind && m.view == viewEdit {
		editorHeight = max(1, editorHeight-2)
	}
	m.editor.SetHeight(editorHeight)
	m.editor.ScrollMargin = m.cfg.UI.EditorScrollMargin
	m.editor.Typewriter = m.cfg.UI.EditorTypewriter
	m.preview.Width = contentWidth(mainStyle, mainW)
	m.preview.Height = contentHeight(mainStyle, innerH)
	m.helpView.Width = contentWidth(m.styles.panel, m.width)
	m.helpView.Height = contentHeight(m.styles.panel, innerH)
	m.markdown.Resize(m.preview.Width)
	if m.view == viewRender {
		m.renderPreview()
	}
}

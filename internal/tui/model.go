package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/llm"
	"github.com/bprendie/weazlwrite/internal/storage"
)

type mode int

const (
	modeVault mode = iota
	modeWrite
	modeAI
	modeGenerating
	modeSaveFile
	modeSaveVault
)

type focus int

const (
	focusTree focus = iota
	focusEditor
	focusPreview
)

type viewMode int

const (
	viewEdit viewMode = iota
	viewRender
)

type model struct {
	cfg         config.Config
	cfgPath     string
	store       *storage.Store
	styles      styles
	mode        mode
	focus       focus
	view        viewMode
	treeVisible bool
	width       int
	height      int
	password    textinput.Model
	aiPrompt    textinput.Model
	filePrompt  textinput.Model
	vaultPrompt textinput.Model
	editor      textarea.Model
	preview     viewport.Model
	markdown    markdownRenderer
	tree        []treeEntry
	treeIdx     int
	cwd         string
	filePath    string
	diskPath    string
	vaultPath   string
	vaultID     string
	isVault     bool
	dirty       bool
	aiBusy      bool
	err         string
	status      string
}

type aiResultMsg struct {
	block string
	err   error
}

func New(cfg config.Config, cfgPath string, store *storage.Store, openPath string) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "database password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.CharLimit = 4096

	ai := textinput.New()
	ai.Placeholder = "insert a basic python loop function"
	ai.CharLimit = 4096

	filePrompt := textinput.New()
	filePrompt.Placeholder = "./notes/document.md"
	filePrompt.CharLimit = 4096

	vaultPrompt := textinput.New()
	vaultPrompt.Placeholder = "projects/specs/document.md"
	vaultPrompt.CharLimit = 4096

	ta := textarea.New()
	ta.Placeholder = "# Untitled\n\nStart writing..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.Focus()

	cwd, _ := os.Getwd()
	m := model{
		cfg:         cfg,
		cfgPath:     cfgPath,
		store:       store,
		styles:      newStyles(),
		mode:        modeVault,
		focus:       focusEditor,
		view:        viewEdit,
		treeVisible: true,
		password:    ti,
		aiPrompt:    ai,
		filePrompt:  filePrompt,
		vaultPrompt: vaultPrompt,
		editor:      ta,
		preview:     viewport.New(0, 0),
		markdown:    markdownRenderer{enabled: cfg.UI.MarkdownEnabled(), style: cfg.UI.MarkdownStyle},
		cwd:         cwd,
		filePath:    openPath,
		status:      "private markdown vault",
	}
	return m
}

func (m model) Init() tea.Cmd {
	has, err := m.store.HasVault()
	if err != nil {
		m.err = err.Error()
	}
	if !has {
		m.password.Placeholder = "create vault password"
		m.status = "create encrypted markdown vault"
		return textinput.Blink
	}
	m.status = "unlock encrypted markdown vault"
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.renderPreview()
	case tea.MouseMsg:
		if m.mode == modeWrite {
			return m.updateMouse(msg)
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == modeVault {
			return m.updateVault(msg)
		}
		if m.mode == modeAI {
			return m.updateAI(msg)
		}
		if m.mode == modeSaveFile {
			return m.updateSaveFile(msg)
		}
		if m.mode == modeSaveVault {
			return m.updateSaveVault(msg)
		}
		if m.mode == modeGenerating {
			return m, nil
		}
		return m.updateWrite(msg)
	case aiResultMsg:
		m.aiBusy = false
		m.mode = modeWrite
		m.setView(viewEdit)
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "ai insert failed"
			return m, nil
		}
		block := strings.TrimSpace(msg.block)
		if block == "" {
			m.err = "ai returned an empty block"
			m.status = "ai insert failed"
			return m, nil
		}
		m.editor.InsertString("\n\n" + block + "\n\n")
		m.dirty = true
		m.err = ""
		m.status = "inserted ai block"
		m.renderPreview()
		return m, nil
	}
	return m, nil
}

func (m model) updateVault(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		password := m.password.Value()
		if strings.TrimSpace(password) == "" {
			m.err = "password is required"
			return m, nil
		}
		has, err := m.store.HasVault()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if has {
			err = m.store.Unlock(password)
		} else {
			err = m.store.CreateVault(password)
		}
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.password.SetValue("")
		m.err = ""
		if err := m.afterUnlock(); err != nil {
			m.err = err.Error()
		}
		m.renderPreview()
		return m, nil
	default:
		var cmd tea.Cmd
		m.password, cmd = m.password.Update(msg)
		return m, cmd
	}
}

func (m model) updateWrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.cycleFocus()
		return m, nil
	case "ctrl+s":
		m.save()
		m.renderTree()
		return m, nil
	case "ctrl+v":
		return m.startSaveVault()
	case "ctrl+f":
		return m.startSaveFile()
	case "ctrl+n":
		m.newVaultNote()
		return m, nil
	case "ctrl+p", "alt+i":
		return m.startAIInsert()
	case "ctrl+o":
		m.toggleTree()
		return m, nil
	case "ctrl+e":
		m.setView(viewEdit)
		return m, nil
	case "ctrl+r":
		m.setView(viewRender)
		return m, nil
	case "pgup":
		if m.focus == focusPreview {
			m.preview.PageUp()
			return m, nil
		}
	case "pgdown":
		if m.focus == focusPreview {
			m.preview.PageDown()
			return m, nil
		}
	case "home":
		if m.focus == focusPreview {
			m.preview.GotoTop()
			return m, nil
		}
	case "end":
		if m.focus == focusPreview {
			m.preview.GotoBottom()
			return m, nil
		}
	case "esc":
		m.setMainFocus()
		return m, nil
	}

	if m.focus == focusTree {
		return m.updateTree(msg)
	}
	if m.view == viewEdit && m.focus == focusEditor {
		before := m.editor.Value()
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		if m.editor.Value() != before {
			m.dirty = true
			m.renderPreview()
		}
		return m, cmd
	}
	if m.view == viewRender {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	mouse := tea.MouseEvent(msg)
	if msg.Action == tea.MouseActionPress && !mouse.IsWheel() {
		m.setFocus(m.focusAtX(msg.X))
		return m, nil
	}
	if !mouse.IsWheel() {
		return m, nil
	}

	target := m.focusAtX(msg.X)
	if target == focusTree {
		switch msg.Type {
		case tea.MouseWheelUp:
			if m.treeIdx > 0 {
				m.treeIdx--
			}
		case tea.MouseWheelDown:
			if m.treeIdx < len(m.tree)-1 {
				m.treeIdx++
			}
		}
		return m, nil
	}
	if target == focusEditor && m.view == viewEdit {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	if target == focusPreview {
		switch msg.Type {
		case tea.MouseWheelUp:
			m.preview.ScrollUp(3)
		case tea.MouseWheelDown:
			m.preview.ScrollDown(3)
		default:
			var cmd tea.Cmd
			m.preview, cmd = m.preview.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) cycleFocus() {
	if !m.treeVisible {
		m.setMainFocus()
		return
	}
	if m.focus == focusTree {
		m.setMainFocus()
		return
	}
	m.setFocus(focusTree)
}

func (m *model) setFocus(f focus) {
	m.focus = f
	if f == focusEditor {
		m.editor.Focus()
		return
	}
	m.editor.Blur()
}

func (m *model) setMainFocus() {
	if m.view == viewEdit {
		m.setFocus(focusEditor)
		return
	}
	m.setFocus(focusPreview)
}

func (m *model) setView(v viewMode) {
	m.view = v
	m.setMainFocus()
	if v == viewRender {
		m.renderPreview()
	}
}

func (m *model) toggleTree() {
	m.treeVisible = !m.treeVisible
	if m.treeVisible {
		m.setFocus(focusTree)
		return
	}
	m.setMainFocus()
}

func (m model) focusAtX(x int) focus {
	treeW, _ := m.layoutWidths()
	if m.treeVisible && x < treeW {
		return focusTree
	}
	if m.view == viewEdit {
		return focusEditor
	}
	return focusPreview
}

func (m model) startAIInsert() (tea.Model, tea.Cmd) {
	m.mode = modeAI
	m.aiPrompt.SetValue("")
	m.aiPrompt.Focus()
	m.editor.Blur()
	m.status = "ai intelligence prompt"
	return m, textinput.Blink
}

func (m model) updateAI(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		instruction := strings.TrimSpace(m.aiPrompt.Value())
		if instruction == "" {
			m.err = "ai prompt is required"
			return m, nil
		}
		m.mode = modeGenerating
		m.aiBusy = true
		m.err = ""
		m.status = "ai generating block"
		return m, m.generateAIBlock(instruction, m.editor.Value())
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "ai insert cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.aiPrompt, cmd = m.aiPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) startSaveFile() (tea.Model, tea.Cmd) {
	path := m.diskPath
	if path == "" {
		name := m.vaultPath
		if name == "" {
			name = m.filePath
		}
		if name == "" {
			name = strings.ToLower(strings.ReplaceAll(titleFor("", m.editor.Value()), " ", "-")) + ".md"
		}
		path = filepath.Join(m.cwd, filepath.Base(name))
	}
	m.mode = modeSaveFile
	m.filePrompt.SetValue(path)
	m.filePrompt.Focus()
	m.editor.Blur()
	m.status = "save to filesystem"
	return m, textinput.Blink
}

func (m model) startSaveVault() (tea.Model, tea.Cmd) {
	path := m.vaultPath
	if path == "" {
		path = defaultVaultPath(m.diskPath, m.cwd, m.editor.Value())
	}
	m.mode = modeSaveVault
	m.vaultPrompt.SetValue(path)
	m.vaultPrompt.Focus()
	m.editor.Blur()
	m.status = "save to encrypted vault"
	return m, textinput.Blink
}

func (m model) updateSaveFile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.filePrompt.Value())
		if path == "" {
			m.err = "filesystem path is required"
			return m, nil
		}
		if err := m.saveToDiskPath(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "filesystem save cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.filePrompt, cmd = m.filePrompt.Update(msg)
		return m, cmd
	}
}

func (m model) updateSaveVault(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.vaultPrompt.Value())
		if path == "" {
			m.err = "vault path is required"
			return m, nil
		}
		if err := m.saveToVaultPath(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "vault save cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.vaultPrompt, cmd = m.vaultPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) generateAIBlock(instruction, document string) tea.Cmd {
	provider := m.cfg.Active()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		block, err := llm.New(provider).GenerateBlock(ctx, document, instruction)
		return aiResultMsg{block: block, err: err}
	}
}

func (m model) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.treeIdx > 0 {
			m.treeIdx--
		}
	case "down", "j":
		if m.treeIdx < len(m.tree)-1 {
			m.treeIdx++
		}
	case "enter":
		m.openSelected()
	case "esc":
		m.setMainFocus()
	}
	return m, nil
}

func (m *model) afterUnlock() error {
	if err := m.renderTree(); err != nil {
		return err
	}
	if m.filePath != "" {
		return m.openDiskPath(m.filePath)
	}
	m.newVaultNote()
	return nil
}

func (m *model) newVaultNote() {
	name := "untitled-" + uuid.NewString()[:8] + ".md"
	m.vaultID = uuid.NewString()
	m.filePath = name
	m.vaultPath = name
	m.diskPath = ""
	m.isVault = true
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + name
	m.setView(viewEdit)
	m.renderPreview()
}

func (m *model) openSelected() {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return
	}
	entry := m.tree[m.treeIdx]
	if entry.isDir && !entry.vault {
		m.cwd = entry.path
		m.renderTree()
		return
	}
	if entry.vault && !entry.isDir {
		if err := m.openVaultPath(entry.path); err != nil {
			m.err = err.Error()
		}
		return
	}
	if !entry.isDir {
		if err := m.openDiskPath(entry.path); err != nil {
			m.err = err.Error()
		}
	}
}

func (m *model) openDiskPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err == nil && info.IsDir() {
		m.cwd = abs
		return m.renderTree()
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	b, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		b = []byte("# " + strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)) + "\n\n")
	} else if err != nil {
		return err
	}
	m.filePath = abs
	m.diskPath = abs
	m.vaultPath = ""
	m.cwd = filepath.Dir(abs)
	m.isVault = false
	m.vaultID = ""
	m.editor.SetValue(string(b))
	m.dirty = false
	m.status = "editing " + abs
	m.err = ""
	m.setView(viewEdit)
	if err := m.renderTree(); err != nil {
		return err
	}
	m.renderPreview()
	return nil
}

func (m *model) openVaultPath(path string) error {
	note, content, ok, err := m.store.LoadNote(path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("vault note not found: %s", path)
	}
	m.filePath = note.Path
	m.vaultPath = note.Path
	m.diskPath = ""
	m.vaultID = note.ID
	m.isVault = true
	m.editor.SetValue(content)
	m.dirty = false
	m.status = "editing vault:" + note.Path
	m.err = ""
	m.setView(viewEdit)
	m.renderPreview()
	return nil
}

func (m *model) save() {
	if m.isVault {
		m.saveToVault()
		return
	}
	if m.diskPath == "" {
		m.err = "no filesystem path; use ctrl+f"
		return
	}
	if err := m.saveToDiskPath(m.diskPath); err != nil {
		m.err = err.Error()
		return
	}
}

func (m *model) saveToVault() {
	path := m.vaultPath
	if path == "" {
		path = defaultVaultPath(m.diskPath, m.cwd, m.editor.Value())
	}
	if err := m.saveToVaultPath(path); err != nil {
		m.err = err.Error()
	}
}

func (m *model) saveToVaultPath(path string) error {
	content := m.editor.Value()
	if m.vaultID == "" {
		m.vaultID = uuid.NewString()
	}
	path = cleanVaultPath(path)
	if path == "" {
		return fmt.Errorf("invalid vault path")
	}
	if err := m.store.SaveNote(m.vaultID, path, titleFor(path, content), content); err != nil {
		return err
	}
	m.vaultPath = path
	m.filePath = path
	m.isVault = true
	m.status = "saved vault:" + path
	m.dirty = false
	m.err = ""
	return nil
}

func (m *model) saveToDiskPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, []byte(m.editor.Value()), 0o644); err != nil {
		return err
	}
	m.filePath = abs
	m.diskPath = abs
	m.isVault = false
	m.cwd = filepath.Dir(abs)
	m.status = "saved " + abs
	m.dirty = false
	m.err = ""
	return nil
}

func defaultVaultPath(diskPath, cwd, content string) string {
	if diskPath != "" {
		if rel, err := filepath.Rel(cwd, diskPath); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			return cleanVaultPath(rel)
		}
		return cleanVaultPath(filepath.Base(diskPath))
	}
	title := strings.ToLower(strings.ReplaceAll(titleFor("", content), " ", "-"))
	if title == "" {
		title = "untitled"
	}
	if filepath.Ext(title) == "" {
		title += ".md"
	}
	return cleanVaultPath(title)
}

func (m *model) renderTree() error {
	notes, err := m.store.ListNotes()
	if err != nil {
		return err
	}
	vaultNotes := make([]string, 0, len(notes))
	for _, note := range notes {
		vaultNotes = append(vaultNotes, note.Path)
	}
	tree, err := readTree(m.cwd, m.cfg.Vault.Root, vaultNotes)
	m.tree = tree
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = max(0, len(m.tree)-1)
	}
	return err
}

func (m *model) renderPreview() {
	content := m.editor.Value()
	width := max(10, m.preview.Width)
	m.preview.SetContent(m.markdown.Render(content, width))
}

func (m *model) resize() {
	innerH := m.bodyHeight()
	_, mainW := m.layoutWidths()
	m.editor.SetWidth(contentWidth(m.styles.panel, mainW))
	m.editor.SetHeight(contentHeight(m.styles.panel, innerH))
	m.preview.Width = contentWidth(m.styles.panel, mainW)
	m.preview.Height = contentHeight(m.styles.panel, innerH)
	m.markdown.Resize(m.preview.Width)
}

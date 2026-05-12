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
)

type focus int

const (
	focusTree focus = iota
	focusEditor
	focusPreview
)

type model struct {
	cfg      config.Config
	cfgPath  string
	store    *storage.Store
	styles   styles
	mode     mode
	focus    focus
	width    int
	height   int
	password textinput.Model
	aiPrompt textinput.Model
	editor   textarea.Model
	preview  viewport.Model
	markdown markdownRenderer
	tree     []treeEntry
	treeIdx  int
	cwd      string
	filePath string
	vaultID  string
	isVault  bool
	dirty    bool
	aiBusy   bool
	err      string
	status   string
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

	ta := textarea.New()
	ta.Placeholder = "# Untitled\n\nStart writing..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.Focus()

	cwd, _ := os.Getwd()
	m := model{
		cfg:      cfg,
		cfgPath:  cfgPath,
		store:    store,
		styles:   newStyles(),
		mode:     modeVault,
		focus:    focusEditor,
		password: ti,
		aiPrompt: ai,
		editor:   ta,
		preview:  viewport.New(0, 0),
		markdown: markdownRenderer{enabled: cfg.UI.MarkdownEnabled(), style: cfg.UI.MarkdownStyle},
		cwd:      cwd,
		filePath: openPath,
		status:   "private markdown vault",
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
		if m.mode == modeGenerating {
			return m, nil
		}
		return m.updateWrite(msg)
	case aiResultMsg:
		m.aiBusy = false
		m.mode = modeWrite
		m.focus = focusEditor
		m.editor.Focus()
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
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil
	case "ctrl+s":
		m.save()
		m.renderTree()
		return m, nil
	case "ctrl+n":
		m.newVaultNote()
		return m, nil
	case "ctrl+p", "alt+i":
		return m.startAIInsert()
	case "ctrl+o":
		m.focus = focusTree
		m.editor.Blur()
		return m, nil
	case "ctrl+e":
		m.focus = focusEditor
		m.editor.Focus()
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
		m.focus = focusEditor
		m.editor.Focus()
		return m, nil
	}

	if m.focus == focusTree {
		return m.updateTree(msg)
	}
	if m.focus == focusEditor {
		before := m.editor.Value()
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		if m.editor.Value() != before {
			m.dirty = true
			m.renderPreview()
		}
		return m, cmd
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
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
	if target == focusEditor {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

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
	return m, nil
}

func (m *model) cycleFocus(delta int) {
	next := int(m.focus) + delta
	if next < int(focusTree) {
		next = int(focusPreview)
	}
	if next > int(focusPreview) {
		next = int(focusTree)
	}
	m.setFocus(focus(next))
}

func (m *model) setFocus(f focus) {
	m.focus = f
	if f == focusEditor {
		m.editor.Focus()
		return
	}
	m.editor.Blur()
}

func (m model) focusAtX(x int) focus {
	treeW, editorW, _ := m.panelWidths()
	if x < treeW {
		return focusTree
	}
	if x < treeW+editorW {
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
		m.focus = focusEditor
		m.editor.Focus()
		m.status = "ai insert cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.aiPrompt, cmd = m.aiPrompt.Update(msg)
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
		m.focus = focusEditor
		m.editor.Focus()
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
	m.isVault = true
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + name
	m.focus = focusEditor
	m.editor.Focus()
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
	m.cwd = filepath.Dir(abs)
	m.isVault = false
	m.vaultID = ""
	m.editor.SetValue(string(b))
	m.dirty = false
	m.status = "editing " + abs
	m.err = ""
	m.focus = focusEditor
	m.editor.Focus()
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
	m.vaultID = note.ID
	m.isVault = true
	m.editor.SetValue(content)
	m.dirty = false
	m.status = "editing vault:" + note.Path
	m.err = ""
	m.focus = focusEditor
	m.editor.Focus()
	m.renderPreview()
	return nil
}

func (m *model) save() {
	content := m.editor.Value()
	if m.isVault {
		if m.vaultID == "" {
			m.vaultID = uuid.NewString()
		}
		if strings.TrimSpace(m.filePath) == "" {
			m.filePath = strings.ToLower(strings.ReplaceAll(titleFor("", content), " ", "-")) + ".md"
		}
		if err := m.store.SaveNote(m.vaultID, m.filePath, titleFor(m.filePath, content), content); err != nil {
			m.err = err.Error()
			return
		}
		m.status = "saved vault:" + m.filePath
		m.dirty = false
		m.err = ""
		return
	}
	if m.filePath == "" {
		m.isVault = true
		m.save()
		return
	}
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		m.err = err.Error()
		return
	}
	if err := os.WriteFile(m.filePath, []byte(content), 0o644); err != nil {
		m.err = err.Error()
		return
	}
	m.status = "saved " + m.filePath
	m.dirty = false
	m.err = ""
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
	_, editorW, previewW := m.panelWidths()
	m.editor.SetWidth(contentWidth(m.styles.panel, editorW))
	m.editor.SetHeight(contentHeight(m.styles.panel, innerH))
	m.preview.Width = contentWidth(m.styles.panel, previewW)
	m.preview.Height = contentHeight(m.styles.panel, innerH)
	m.markdown.Resize(m.preview.Width)
}

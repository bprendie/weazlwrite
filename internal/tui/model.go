package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/importer"
	"github.com/bprendie/weazlwrite/internal/llm"
	"github.com/bprendie/weazlwrite/internal/storage"
)

type mode int

const (
	modeVaultPicker mode = iota
	modeVaultName
	modeVault
	modeVaultConfirm
	modeWrite
	modeAI
	modeGenerating
	modeSaveFile
	modeSaveVault
	modeNewFolder
	modeConfirmDelete
	modeConfirmEyesOff
	modeRenameTree
	modeHelp
	modeFind
	modeJumpPage
	modeImporting
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
	cfg           config.Config
	cfgPath       string
	store         *storage.Store
	styles        styles
	mode          mode
	focus         focus
	view          viewMode
	treeVisible   bool
	mouseCapture  bool
	width         int
	height        int
	password      textinput.Model
	confirmPass   textinput.Model
	vaultName     textinput.Model
	aiPrompt      textinput.Model
	filePrompt    textinput.Model
	vaultPrompt   textinput.Model
	folderPrompt  textinput.Model
	renamePrompt  textinput.Model
	findPrompt    textinput.Model
	jumpPrompt    textinput.Model
	working       spinner.Model
	editor        textarea.Model
	preview       viewport.Model
	helpView      viewport.Model
	markdown      markdownRenderer
	tree          []treeEntry
	treeIdx       int
	treeOffset    int
	treeExpanded  map[string]bool
	eyesOnlyPaths map[string]bool
	vaults        []vaultChoice
	vaultIdx      int
	activeVault   vaultChoice
	deleteTarget  treeEntry
	eyesOffTarget treeEntry
	renameTarget  treeEntry
	carryTarget   treeEntry
	cwd           string
	filePath      string
	diskPath      string
	vaultPath     string
	vaultID       string
	isVault       bool
	eyesOnly      bool
	dirty         bool
	aiBusy        bool
	generatingAt  time.Time
	lastFind      string
	pendingPass   string
	err           string
	status        string
}

type vaultChoice struct {
	name   string
	path   string
	exists bool
}

type aiResultMsg struct {
	block string
	err   error
}

type importResultMsg struct {
	files    int
	folders  int
	warnings int
	err      error
}

func New(cfg config.Config, cfgPath string, openPath string) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "database password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.CharLimit = 4096

	confirmPass := textinput.New()
	confirmPass.Placeholder = "confirm vault password"
	confirmPass.EchoMode = textinput.EchoPassword
	confirmPass.CharLimit = 4096

	vaultName := textinput.New()
	vaultName.Placeholder = "work"
	vaultName.CharLimit = 80

	ai := textinput.New()
	ai.Placeholder = "insert a basic python loop function"
	ai.CharLimit = 4096

	filePrompt := textinput.New()
	filePrompt.Placeholder = "./notes/document.md"
	filePrompt.CharLimit = 4096

	vaultPrompt := textinput.New()
	vaultPrompt.Placeholder = "projects/specs/document.md"
	vaultPrompt.CharLimit = 4096

	folderPrompt := textinput.New()
	folderPrompt.Placeholder = "folder name"
	folderPrompt.CharLimit = 4096

	renamePrompt := textinput.New()
	renamePrompt.Placeholder = "new path"
	renamePrompt.CharLimit = 4096

	findPrompt := textinput.New()
	findPrompt.Placeholder = "find text"
	findPrompt.CharLimit = 4096

	jumpPrompt := textinput.New()
	jumpPrompt.Placeholder = "page number"
	jumpPrompt.CharLimit = 64

	s := newStyles()
	working := spinner.New(
		spinner.WithSpinner(spinner.Jump),
		spinner.WithStyle(s.status),
	)

	ta := textarea.New()
	ta.Placeholder = "# Untitled\n\nStart writing..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.Focus()

	cwd, _ := os.Getwd()
	m := model{
		cfg:          cfg,
		cfgPath:      cfgPath,
		styles:       s,
		mode:         modeVaultPicker,
		focus:        focusEditor,
		view:         viewEdit,
		treeVisible:  true,
		mouseCapture: true,
		password:     ti,
		confirmPass:  confirmPass,
		vaultName:    vaultName,
		aiPrompt:     ai,
		filePrompt:   filePrompt,
		vaultPrompt:  vaultPrompt,
		folderPrompt: folderPrompt,
		renamePrompt: renamePrompt,
		findPrompt:   findPrompt,
		jumpPrompt:   jumpPrompt,
		working:      working,
		editor:       ta,
		preview:      viewport.New(0, 0),
		helpView:     viewport.New(0, 0),
		markdown:     markdownRenderer{enabled: cfg.UI.MarkdownEnabled(), style: cfg.UI.MarkdownStyle},
		treeExpanded: map[string]bool{
			"vault:": true,
			"file:":  true,
		},
		cwd:      cwd,
		filePath: openPath,
		status:   "private markdown vault",
	}
	if err := m.refreshVaultChoices(); err != nil {
		m.err = err.Error()
	}
	m.status = "select vault"
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.EnableMouseCellMotion)
}

func (m *model) prepareVaultPassword() {
	has, err := m.store.HasVault()
	if err != nil {
		m.err = err.Error()
	}
	if !has {
		m.password.Placeholder = "create vault password"
		m.status = "create encrypted markdown vault: " + m.activeVault.name
		return
	}
	m.password.Placeholder = "vault password"
	m.status = "unlock encrypted markdown vault: " + m.activeVault.name
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.renderPreview()
		m.renderHelp()
	case tea.MouseMsg:
		if m.mode == modeWrite && m.mouseCapture {
			return m.updateMouse(msg)
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == modeVaultPicker {
			return m.updateVaultPicker(msg)
		}
		if m.mode == modeVaultName {
			return m.updateVaultName(msg)
		}
		if m.mode == modeVault {
			return m.updateVault(msg)
		}
		if m.mode == modeVaultConfirm {
			return m.updateVaultConfirm(msg)
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
		if m.mode == modeNewFolder {
			return m.updateNewFolder(msg)
		}
		if m.mode == modeConfirmDelete {
			return m.updateConfirmDelete(msg)
		}
		if m.mode == modeConfirmEyesOff {
			return m.updateConfirmEyesOff(msg)
		}
		if m.mode == modeRenameTree {
			return m.updateRenameTree(msg)
		}
		if m.mode == modeHelp {
			return m.updateHelp(msg)
		}
		if m.mode == modeFind {
			return m.updateFind(msg)
		}
		if m.mode == modeJumpPage {
			return m.updateJumpPage(msg)
		}
		if m.mode == modeImporting {
			return m, nil
		}
		if m.mode == modeGenerating {
			return m, nil
		}
		return m.updateWrite(msg)
	case aiResultMsg:
		m.aiBusy = false
		m.generatingAt = time.Time{}
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
	case importResultMsg:
		m.mode = modeWrite
		m.aiBusy = false
		m.generatingAt = time.Time{}
		m.focus = focusTree
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "import failed"
			return m, nil
		}
		m.err = ""
		m.status = fmt.Sprintf("imported %d files and %d folders to vault", msg.files, msg.folders)
		if msg.warnings > 0 {
			m.status += fmt.Sprintf("; skipped %d image-based files", msg.warnings)
		}
		if err := m.renderTree(); err != nil {
			m.err = err.Error()
		}
		return m, nil
	case spinner.TickMsg:
		if m.mode == modeGenerating || m.mode == modeImporting || m.aiBusy {
			var cmd tea.Cmd
			m.working, cmd = m.working.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) updateVaultPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.vaultIdx > 0 {
			m.vaultIdx--
		}
	case "down", "j":
		if m.vaultIdx < len(m.vaults)-1 {
			m.vaultIdx++
		}
	case "enter":
		if len(m.vaults) == 0 {
			return m.startVaultName()
		}
		if err := m.selectVault(m.vaults[m.vaultIdx]); err != nil {
			m.err = err.Error()
			return m, nil
		}
		return m, textinput.Blink
	case "n":
		return m.startVaultName()
	}
	return m, nil
}

func (m model) startVaultName() (tea.Model, tea.Cmd) {
	m.mode = modeVaultName
	m.vaultName.SetValue("")
	m.vaultName.Focus()
	m.status = "new vault"
	m.err = ""
	return m, textinput.Blink
}

func (m model) updateVaultName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.vaultName.Value())
		if name == "" {
			m.err = "vault name is required"
			return m, nil
		}
		choice, err := m.newVaultChoice(name)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if err := m.selectVault(choice); err != nil {
			m.err = err.Error()
			return m, nil
		}
		return m, textinput.Blink
	case "esc":
		m.mode = modeVaultPicker
		m.status = "select vault"
		return m, nil
	default:
		var cmd tea.Cmd
		m.vaultName, cmd = m.vaultName.Update(msg)
		return m, cmd
	}
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
			m.pendingPass = password
			m.password.SetValue("")
			m.confirmPass.SetValue("")
			m.confirmPass.Focus()
			m.mode = modeVaultConfirm
			m.status = "confirm encrypted markdown vault password"
			m.err = ""
			return m, textinput.Blink
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

func (m model) updateVaultConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		confirm := m.confirmPass.Value()
		if strings.TrimSpace(confirm) == "" {
			m.err = "password confirmation is required"
			return m, nil
		}
		if confirm != m.pendingPass {
			m.err = "passwords do not match"
			m.confirmPass.SetValue("")
			return m, nil
		}
		if err := m.store.CreateVault(m.pendingPass); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.pendingPass = ""
		m.confirmPass.SetValue("")
		m.mode = modeWrite
		m.err = ""
		if err := m.afterUnlock(); err != nil {
			m.err = err.Error()
		}
		m.renderPreview()
		return m, nil
	case "esc":
		m.pendingPass = ""
		m.confirmPass.SetValue("")
		m.password.SetValue("")
		m.password.Focus()
		m.mode = modeVault
		m.prepareVaultPassword()
		m.status = "vault creation cancelled"
		return m, textinput.Blink
	default:
		var cmd tea.Cmd
		m.confirmPass, cmd = m.confirmPass.Update(msg)
		return m, cmd
	}
}

func (m *model) refreshVaultChoices() error {
	if err := os.MkdirAll(m.cfg.Vault.Root, 0o700); err != nil {
		return err
	}
	seen := map[string]bool{}
	var choices []vaultChoice
	add := func(path string) {
		clean, err := filepath.Abs(path)
		if err == nil {
			path = clean
		}
		if seen[path] {
			return
		}
		seen[path] = true
		choices = append(choices, vaultChoice{
			name:   vaultNameFromPath(path),
			path:   path,
			exists: true,
		})
	}

	if err := filepath.WalkDir(m.cfg.Vault.Root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != m.cfg.Vault.Root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".sqlite3") {
			add(path)
		}
		return nil
	}); err != nil {
		return err
	}

	sort.Slice(choices, func(i, j int) bool {
		return strings.ToLower(choices[i].name) < strings.ToLower(choices[j].name)
	})
	activePath, _ := filepath.Abs(m.cfg.Database.Path)
	m.vaultIdx = 0
	for i, choice := range choices {
		if choice.path == activePath {
			m.vaultIdx = i
			break
		}
	}
	if len(choices) == 0 {
		choices = append(choices, vaultChoice{
			name:   vaultNameFromPath(m.cfg.Database.Path),
			path:   m.cfg.Database.Path,
			exists: false,
		})
	}
	m.vaults = choices
	if m.vaultIdx >= len(m.vaults) {
		m.vaultIdx = max(0, len(m.vaults)-1)
	}
	return nil
}

func (m *model) selectVault(choice vaultChoice) error {
	if m.store != nil {
		_ = m.store.Close()
		m.store = nil
	}
	store, err := storage.Open(choice.path)
	if err != nil {
		return err
	}
	if err := store.Migrate(); err != nil {
		store.Close()
		return err
	}
	m.store = store
	m.activeVault = choice
	m.cfg.Database.Path = choice.path
	if err := config.Save(m.cfgPath, m.cfg); err != nil {
		return err
	}
	m.mode = modeVault
	m.password.SetValue("")
	m.password.Focus()
	m.prepareVaultPassword()
	m.err = ""
	return nil
}

func (m model) newVaultChoice(name string) (vaultChoice, error) {
	slug := vaultSlug(name)
	if slug == "" {
		return vaultChoice{}, fmt.Errorf("vault name must contain a letter or number")
	}
	path := filepath.Join(m.cfg.Vault.Root, slug+".sqlite3")
	if _, err := os.Stat(path); err == nil {
		return vaultChoice{}, fmt.Errorf("vault already exists: %s", slug)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return vaultChoice{}, err
	}
	return vaultChoice{name: slug, path: path}, nil
}

func vaultNameFromPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "weazlwrite" {
		return "default"
	}
	return name
}

func vaultSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
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
	case "alt+f":
		return m.startSaveFile()
	case "ctrl+f":
		return m.startFind()
	case "ctrl+g":
		return m.startJumpPage()
	case "alt+o":
		return m.toggleCurrentEyesOnly()
	case "ctrl+n":
		m.newVaultNote()
		return m, nil
	case "ctrl+p", "alt+i":
		return m.startAIInsert()
	case "ctrl+o":
		m.toggleTree()
		return m, nil
	case "ctrl+y":
		return m.toggleMouseCapture()
	case "ctrl+k", "?", "h", "f1":
		return m.startHelp()
	case "ctrl+e":
		m.setView(viewEdit)
		return m, nil
	case "ctrl+r":
		m.setView(viewRender)
		return m, nil
	case "pgup":
		if m.focus == focusTree {
			m.pageTree(-1)
			return m, nil
		}
		if m.focus == focusPreview {
			m.preview.PageUp()
			return m, nil
		}
		if m.focus == focusEditor {
			m.editorPageUp()
			return m, nil
		}
	case "pgdown":
		if m.focus == focusTree {
			m.pageTree(1)
			return m, nil
		}
		if m.focus == focusPreview {
			m.preview.PageDown()
			return m, nil
		}
		if m.focus == focusEditor {
			m.editorPageDown()
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
			m.scrollTree(-3)
		case tea.MouseWheelDown:
			m.scrollTree(3)
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
	} else {
		m.editor.Blur()
	}
	m.resize()
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

func (m model) toggleMouseCapture() (tea.Model, tea.Cmd) {
	if m.eyesOnly {
		m.mouseCapture = true
		m.err = "eyes only notes keep copy protection on"
		return m, tea.EnableMouseCellMotion
	}
	m.mouseCapture = !m.mouseCapture
	if m.mouseCapture {
		m.status = "mouse capture on"
		return m, tea.EnableMouseCellMotion
	}
	m.status = "mouse capture off; terminal selection enabled"
	return m, tea.DisableMouse
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
		m.generatingAt = time.Now()
		m.err = ""
		m.status = "ai generating block"
		return m, tea.Batch(m.generateAIBlock(instruction, m.editor.Value()), m.working.Tick)
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
	m.status = "save to: filesystem"
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
	m.status = "save to: encrypted vault"
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
		m.renderTree()
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

func (m model) startNewFolder() (tea.Model, tea.Cmd) {
	if m.focus != focusTree {
		m.focus = focusTree
	}
	base := m.folderBasePath()
	m.mode = modeNewFolder
	m.folderPrompt.SetValue(base)
	m.folderPrompt.Focus()
	m.editor.Blur()
	m.status = "new folder"
	return m, textinput.Blink
}

func (m model) updateNewFolder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.folderPrompt.Value())
		if path == "" {
			m.err = "folder path is required"
			return m, nil
		}
		if err := m.createFolder(path); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.status = "new folder cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.folderPrompt, cmd = m.folderPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) startConfirmDelete() (tea.Model, tea.Cmd) {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return m, nil
	}
	entry := m.tree[m.treeIdx]
	if entry.id == "vault:" || entry.id == "file:" {
		m.err = "cannot delete tree root"
		return m, nil
	}
	m.deleteTarget = entry
	m.mode = modeConfirmDelete
	m.status = "confirm delete"
	return m, nil
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if err := m.deleteTreeEntry(m.deleteTarget); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.deleteTarget = treeEntry{}
		m.renderTree()
		return m, nil
	case "n", "N", "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.deleteTarget = treeEntry{}
		m.status = "delete cancelled"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateConfirmEyesOff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		entry := m.eyesOffTarget
		path := cleanVaultPath(entry.path)
		if err := m.store.SetNoteEyesOnly(path, false); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		if m.eyesOnlyPaths != nil {
			delete(m.eyesOnlyPaths, path)
		}
		if m.isVault && cleanVaultPath(m.vaultPath) == path {
			m.eyesOnly = false
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.eyesOffTarget = treeEntry{}
		m.err = ""
		m.status = "eyes only disabled:" + path
		if err := m.renderTree(); err != nil {
			m.err = err.Error()
		}
		return m, nil
	case "n", "N", "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.eyesOffTarget = treeEntry{}
		m.status = "eyes only unchanged"
		return m, nil
	default:
		return m, nil
	}
}

func (m model) startRenameTree() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if entry.id == "vault:" || entry.id == "file:" {
		m.err = "cannot rename tree root"
		return m, nil
	}
	m.renameTarget = entry
	m.mode = modeRenameTree
	m.renamePrompt.SetValue(entry.path)
	m.renamePrompt.Focus()
	m.editor.Blur()
	m.status = "rename/move"
	return m, textinput.Blink
}

func (m model) updateRenameTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.renamePrompt.Value())
		if path == "" {
			m.err = "path is required"
			return m, nil
		}
		if err := m.renameTreeEntry(m.renameTarget, path); err != nil {
			m.err = err.Error()
			m.mode = modeWrite
			m.focus = focusTree
			return m, nil
		}
		m.mode = modeWrite
		m.focus = focusTree
		m.renameTarget = treeEntry{}
		m.renderTree()
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.focus = focusTree
		m.renameTarget = treeEntry{}
		m.status = "rename cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.renamePrompt, cmd = m.renamePrompt.Update(msg)
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
	case "pgup":
		m.pageTree(-1)
	case "pgdown":
		m.pageTree(1)
	case "enter":
		return m, m.openSelected()
	case " ":
		m.pickupOrDropSelected()
	case "n":
		return m.startNewFolder()
	case "d":
		return m.startConfirmDelete()
	case "r":
		return m.startRenameTree()
	case "i":
		return m.startImportSelectedToVault()
	case "o":
		return m.toggleEyesOnlySelected()
	case "esc":
		m.carryTarget = treeEntry{}
		m.setMainFocus()
	}
	m.ensureTreeSelectionVisible()
	return m, nil
}

func (m *model) pageTree(direction int) {
	if len(m.tree) == 0 {
		return
	}
	height := max(1, m.treeContentHeight())
	step := max(1, height-1)
	m.treeIdx = min(max(0, m.treeIdx+direction*step), len(m.tree)-1)
	m.ensureTreeSelectionVisible()
}

func (m *model) scrollTree(delta int) {
	if len(m.tree) == 0 {
		return
	}
	height := max(1, m.treeContentHeight())
	maxOffset := max(0, len(m.tree)-height)
	m.treeOffset = min(max(0, m.treeOffset+delta), maxOffset)
	if m.treeIdx < m.treeOffset {
		m.treeIdx = m.treeOffset
	}
	if m.treeIdx >= m.treeOffset+height {
		m.treeIdx = min(len(m.tree)-1, m.treeOffset+height-1)
	}
}

func (m model) startHelp() (tea.Model, tea.Cmd) {
	m.mode = modeHelp
	m.editor.Blur()
	m.renderHelp()
	m.status = "help"
	return m, nil
}

func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "h", "?":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "private markdown vault"
		return m, nil
	default:
		var cmd tea.Cmd
		m.helpView, cmd = m.helpView.Update(msg)
		return m, cmd
	}
}

func (m model) startFind() (tea.Model, tea.Cmd) {
	m.mode = modeFind
	m.findPrompt.SetValue(m.lastFind)
	m.findPrompt.Focus()
	m.editor.Blur()
	m.status = "find"
	return m, textinput.Blink
}

func (m model) updateFind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.findPrompt.Value())
		if query == "" {
			m.err = "find text is required"
			return m, nil
		}
		m.lastFind = query
		m.mode = modeWrite
		m.setMainFocus()
		m.findNext(query)
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "find cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.findPrompt, cmd = m.findPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) startJumpPage() (tea.Model, tea.Cmd) {
	m.mode = modeJumpPage
	m.jumpPrompt.SetValue(strconv.Itoa(max(1, m.currentPage())))
	m.jumpPrompt.Focus()
	m.editor.Blur()
	m.status = "jump to page"
	return m, textinput.Blink
}

func (m model) updateJumpPage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		page, err := strconv.Atoi(strings.TrimSpace(m.jumpPrompt.Value()))
		if err != nil || page < 1 {
			m.err = "page must be a positive number"
			return m, nil
		}
		m.mode = modeWrite
		m.setMainFocus()
		m.jumpToPage(page)
		return m, nil
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "jump cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.jumpPrompt, cmd = m.jumpPrompt.Update(msg)
		return m, cmd
	}
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
	m.eyesOnly = false
	m.expandTreeTo("vault:" + name)
	m.editor.SetValue("# Untitled\n\n")
	m.dirty = true
	m.status = "new vault note " + name
	m.setView(viewEdit)
	m.renderPreview()
}

func (m *model) openSelected() tea.Cmd {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return nil
	}
	entry := m.tree[m.treeIdx]
	if entry.isDir {
		m.toggleTreeEntry(entry)
		return nil
	}
	if entry.vault && !entry.isDir {
		if err := m.openVaultPath(entry.path); err != nil {
			m.err = err.Error()
		}
		if m.eyesOnly {
			return tea.EnableMouseCellMotion
		}
		return nil
	}
	if !entry.isDir {
		if err := m.openDiskPath(entry.path); err != nil {
			m.err = err.Error()
		}
	}
	return nil
}

func (m *model) toggleSelectedDir() {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return
	}
	entry := m.tree[m.treeIdx]
	if entry.isDir {
		m.toggleTreeEntry(entry)
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) toggleTreeEntry(entry treeEntry) {
	if entry.id == "" {
		return
	}
	m.treeExpanded[entry.id] = !m.treeExpanded[entry.id]
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) pickupOrDropSelected() {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return
	}
	if m.carryTarget.id == "" {
		if entry.id == "vault:" || entry.id == "file:" || entry.isDir {
			m.err = "space picks up files only; press enter to fold folders"
			return
		}
		m.carryTarget = entry
		m.err = ""
		m.status = "picked up " + entry.path
		return
	}
	source := m.carryTarget
	if err := m.dropTreeEntry(source, entry); err != nil {
		m.err = err.Error()
		return
	}
	m.carryTarget = treeEntry{}
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
}

func (m model) selectedTreeEntry() (treeEntry, bool) {
	if len(m.tree) == 0 || m.treeIdx >= len(m.tree) {
		return treeEntry{}, false
	}
	return m.tree[m.treeIdx], true
}

func (m model) folderBasePath() string {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return ""
	}
	if entry.vault {
		if entry.id == "vault:" {
			return ""
		}
		if entry.isDir {
			return strings.TrimSuffix(entry.path, "/") + "/"
		}
		parent := vaultParent(entry.path)
		if parent == "" {
			return ""
		}
		return parent + "/"
	}
	if entry.id == "file:" {
		return m.cwd + string(filepath.Separator)
	}
	if entry.isDir {
		return entry.path + string(filepath.Separator)
	}
	return filepath.Dir(entry.path) + string(filepath.Separator)
}

func (m *model) createFolder(path string) error {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return fmt.Errorf("no tree selection")
	}
	if entry.vault {
		clean := cleanVaultPath(path)
		if clean == "" {
			return fmt.Errorf("invalid vault folder path")
		}
		if err := m.store.SaveFolder(clean); err != nil {
			return err
		}
		m.expandTreeTo("vault:" + clean)
		m.status = "created vault folder:" + clean
		m.err = ""
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	m.expandTreeTo("file:" + abs)
	m.status = "created folder " + abs
	m.err = ""
	return nil
}

func (m *model) deleteTreeEntry(entry treeEntry) error {
	if entry.id == "" || entry.id == "vault:" || entry.id == "file:" {
		return fmt.Errorf("cannot delete tree root")
	}
	if entry.vault {
		if entry.isDir {
			if err := m.store.DeleteFolder(cleanVaultPath(entry.path)); err != nil {
				return err
			}
			delete(m.treeExpanded, entry.id)
			m.status = "deleted vault folder:" + entry.path
			return nil
		}
		if err := m.store.DeleteNote(entry.path); err != nil {
			return err
		}
		if m.isVault && cleanVaultPath(m.vaultPath) == cleanVaultPath(entry.path) {
			m.vaultID = uuid.NewString()
			m.vaultPath = ""
			m.filePath = ""
			m.dirty = true
		}
		m.status = "deleted vault note:" + entry.path
		return nil
	}
	if entry.path == "" {
		return fmt.Errorf("missing filesystem path")
	}
	if err := os.Remove(entry.path); err != nil {
		return err
	}
	if !m.isVault && m.diskPath == entry.path {
		m.diskPath = ""
		m.filePath = ""
		m.dirty = true
	}
	delete(m.treeExpanded, entry.id)
	m.status = "deleted " + entry.path
	return nil
}

func (m model) toggleEyesOnlySelected() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if !entry.vault || entry.isDir || entry.id == "vault:" {
		m.err = "eyes only is available for vault files"
		return m, nil
	}
	return m.toggleEyesOnlyPath(cleanVaultPath(entry.path), entry)
}

func (m model) toggleCurrentEyesOnly() (tea.Model, tea.Cmd) {
	if !m.isVault || strings.TrimSpace(m.vaultPath) == "" {
		m.err = "open a vault note to toggle eyes only"
		return m, nil
	}
	path := cleanVaultPath(m.vaultPath)
	return m.toggleEyesOnlyPath(path, treeEntry{
		id:    "vault:" + path,
		name:  filepath.Base(path),
		path:  path,
		vault: true,
	})
}

func (m model) toggleEyesOnlyPath(path string, entry treeEntry) (tea.Model, tea.Cmd) {
	if path == "" {
		m.err = "missing vault note path"
		return m, nil
	}
	next := !m.eyesOnlyPaths[path]
	if !next {
		entry.path = path
		entry.vault = true
		m.eyesOffTarget = entry
		m.mode = modeConfirmEyesOff
		m.status = "confirm disabling eyes only"
		m.err = ""
		return m, nil
	}
	if err := m.store.SetNoteEyesOnly(path, true); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if m.eyesOnlyPaths == nil {
		m.eyesOnlyPaths = map[string]bool{}
	}
	m.eyesOnlyPaths[path] = true
	if m.isVault && cleanVaultPath(m.vaultPath) == path {
		m.eyesOnly = true
		m.mouseCapture = true
	}
	m.status = "eyes only enabled:" + path
	m.err = ""
	if err := m.renderTree(); err != nil {
		m.err = err.Error()
	}
	if m.isVault && cleanVaultPath(m.vaultPath) == path {
		return m, tea.EnableMouseCellMotion
	}
	return m, nil
}

func (m model) startImportSelectedToVault() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedTreeEntry()
	if !ok {
		return m, nil
	}
	if entry.vault {
		m.err = "select a filesystem file or folder to import"
		return m, nil
	}
	if entry.id == "file:" {
		m.err = "select a filesystem file or folder to import"
		return m, nil
	}
	m.mode = modeImporting
	m.aiBusy = true
	m.generatingAt = time.Now()
	m.err = ""
	m.status = "importing " + entry.path
	m.editor.Blur()
	return m, tea.Batch(m.importFilesystemEntryCmd(entry), m.working.Tick)
}

func (m model) importFilesystemEntryCmd(entry treeEntry) tea.Cmd {
	return func() tea.Msg {
		files, folders, warnings, err := m.importFilesystemEntry(entry)
		return importResultMsg{files: files, folders: folders, warnings: warnings, err: err}
	}
}

func (m *model) importFilesystemEntry(entry treeEntry) (int, int, int, error) {
	info, err := os.Stat(entry.path)
	if err != nil {
		return 0, 0, 0, err
	}
	if !info.IsDir() {
		if !isVaultImportFile(entry.path) {
			return 0, 0, 0, fmt.Errorf("only markdown, text, PDF, and DOCX files can be imported")
		}
		target := importPathForFile(entry.path, m.cwd)
		if err := m.importFileToVault(entry.path, target); err != nil {
			return 0, 0, 0, err
		}
		m.expandTreeTo("vault:" + target)
		return 1, 0, 0, nil
	}

	root := entry.path
	files := 0
	folders := 0
	warnings := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := cleanVaultPath(filepath.ToSlash(rel))
		if target == "" {
			return nil
		}
		if d.IsDir() {
			if err := m.store.SaveFolder(target); err != nil {
				return err
			}
			folders++
			return nil
		}
		if !isVaultImportFile(path) {
			return nil
		}
		if err := m.importFileToVault(path, target); err != nil {
			if errors.Is(err, importer.ErrImageOnlyDocument) {
				warnings++
				return nil
			}
			return err
		}
		files++
		return nil
	})
	if err != nil {
		return files, folders, warnings, err
	}
	m.expandTreeTo("vault:")
	return files, folders, warnings, nil
}

func (m *model) importFileToVault(sourcePath, vaultPath string) error {
	doc, err := importer.Convert(sourcePath)
	if err != nil {
		return err
	}
	return m.store.SaveNote(uuid.NewString(), vaultPath, titleFor(vaultPath, doc.Markdown), doc.Markdown)
}

func (m *model) dropTreeEntry(source, destination treeEntry) error {
	if destination.id == "" {
		return fmt.Errorf("missing destination")
	}
	if !destination.isDir {
		return fmt.Errorf("drop onto a folder")
	}
	if source.vault != destination.vault {
		return fmt.Errorf("cannot move between vault and filesystem; use save to")
	}
	base := entryBaseName(source)
	if base == "" {
		return fmt.Errorf("missing source name")
	}
	if source.vault {
		parent := ""
		if destination.id != "vault:" {
			parent = cleanVaultPath(destination.path)
		}
		next := base
		if parent != "" {
			next = parent + "/" + base
		}
		return m.renameTreeEntry(source, next)
	}
	parent := m.cwd
	if destination.id != "file:" {
		parent = destination.path
	}
	return m.renameTreeEntry(source, filepath.Join(parent, base))
}

func (m *model) renameTreeEntry(entry treeEntry, newPath string) error {
	if entry.id == "" || entry.id == "vault:" || entry.id == "file:" {
		return fmt.Errorf("cannot rename tree root")
	}
	if entry.vault {
		clean := cleanVaultPath(newPath)
		if clean == "" {
			return fmt.Errorf("invalid vault path")
		}
		old := cleanVaultPath(entry.path)
		if entry.isDir {
			if err := m.store.RenameFolder(old, clean); err != nil {
				return err
			}
			m.rewriteCurrentVaultPath(old, clean)
			delete(m.treeExpanded, entry.id)
			m.expandTreeTo("vault:" + clean)
			m.status = "moved vault folder:" + clean
			m.err = ""
			return nil
		}
		if err := m.store.RenameNote(old, clean); err != nil {
			return err
		}
		if m.isVault && cleanVaultPath(m.vaultPath) == old {
			m.vaultPath = clean
			m.filePath = clean
		}
		m.expandTreeTo("vault:" + clean)
		m.status = "renamed vault note:" + clean
		m.err = ""
		return nil
	}
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	if entry.isDir {
		if isPathInside(abs, entry.path) {
			return fmt.Errorf("cannot move a folder inside itself")
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if entry.path == abs {
		return nil
	}
	if err := os.Rename(entry.path, abs); err != nil {
		return err
	}
	m.rewriteCurrentDiskPath(entry.path, abs)
	delete(m.treeExpanded, entry.id)
	m.expandTreeTo("file:" + abs)
	m.status = "moved " + abs
	m.err = ""
	return nil
}

func (m *model) rewriteCurrentVaultPath(oldPath, newPath string) {
	if !m.isVault {
		return
	}
	current := cleanVaultPath(m.vaultPath)
	if current == oldPath || strings.HasPrefix(current, oldPath+"/") {
		m.vaultPath = renamedTreePath(current, oldPath, newPath)
		m.filePath = m.vaultPath
	}
}

func (m *model) rewriteCurrentDiskPath(oldPath, newPath string) {
	if m.isVault || m.diskPath == "" {
		return
	}
	current := filepath.Clean(m.diskPath)
	oldPath = filepath.Clean(oldPath)
	newPath = filepath.Clean(newPath)
	if current == oldPath || strings.HasPrefix(current, oldPath+string(filepath.Separator)) {
		m.diskPath = renamedTreePath(current, oldPath, newPath)
		m.filePath = m.diskPath
		m.cwd = filepath.Dir(m.diskPath)
	}
}

func renamedTreePath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	return newPath + strings.TrimPrefix(path, oldPath)
}

func entryBaseName(entry treeEntry) string {
	if entry.vault {
		return vaultBase(entry.path)
	}
	return filepath.Base(entry.path)
}

func isPathInside(path, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	return path == parent || strings.HasPrefix(path, parent+string(filepath.Separator))
}

func isVaultImportFile(path string) bool {
	return importer.Supported(path) || strings.EqualFold(filepath.Ext(path), ".doc")
}

func importPathForFile(path, cwd string) string {
	path = importer.MarkdownPath(path)
	rel, err := filepath.Rel(cwd, path)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		if clean := cleanVaultPath(filepath.ToSlash(rel)); clean != "" {
			return clean
		}
	}
	return cleanVaultPath(filepath.Base(path))
}

func isConvertibleDocument(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf", ".docx":
		return true
	default:
		return false
	}
}

func (m *model) findNext(query string) {
	if m.view == viewRender {
		m.findInPreview(query)
		return
	}
	m.findInEditor(query)
}

func (m *model) findInPreview(query string) {
	rendered := m.markdown.Render(m.editor.Value(), max(10, m.preview.Width))
	lines := strings.Split(rendered, "\n")
	q := strings.ToLower(query)
	start := min(len(lines), m.preview.YOffset+1)
	for pass := 0; pass < 2; pass++ {
		from := 0
		if pass == 0 {
			from = start
		}
		for i := from; i < len(lines); i++ {
			if strings.Contains(strings.ToLower(lines[i]), q) {
				m.preview.SetYOffset(max(0, i-1))
				m.err = ""
				m.status = fmt.Sprintf("found %q on page %d", query, m.currentPage())
				return
			}
		}
	}
	m.err = "not found: " + query
}

func (m *model) findInEditor(query string) {
	lines := strings.Split(m.editor.Value(), "\n")
	q := strings.ToLower(query)
	start := min(len(lines), m.editor.Line()+1)
	for pass := 0; pass < 2; pass++ {
		from := 0
		if pass == 0 {
			from = start
		}
		for i := from; i < len(lines); i++ {
			col := strings.Index(strings.ToLower(lines[i]), q)
			if col >= 0 {
				m.moveEditorToLine(i)
				m.editor.SetCursor(col)
				m.err = ""
				m.status = fmt.Sprintf("found %q on page %d", query, m.currentPage())
				return
			}
		}
	}
	m.err = "not found: " + query
}

func (m *model) jumpToPage(page int) {
	page = min(max(1, page), max(1, m.totalPages()))
	if m.view == viewRender {
		m.preview.SetYOffset((page - 1) * max(1, m.preview.Height))
	} else {
		m.moveEditorToLine((page - 1) * max(1, m.editor.Height()))
	}
	m.err = ""
	m.status = fmt.Sprintf("page %d of %d", m.currentPage(), m.totalPages())
}

func (m model) currentPage() int {
	if m.view == viewRender {
		return min(max(1, m.preview.YOffset/max(1, m.preview.Height)+1), max(1, m.totalPages()))
	}
	return min(max(1, m.editor.Line()/max(1, m.editor.Height())+1), max(1, m.totalPages()))
}

func (m model) totalPages() int {
	if m.view == viewRender {
		return max(1, (m.preview.TotalLineCount()+max(1, m.preview.Height)-1)/max(1, m.preview.Height))
	}
	return max(1, (m.editor.LineCount()+max(1, m.editor.Height())-1)/max(1, m.editor.Height()))
}

func (m *model) moveEditorToLine(line int) {
	line = min(max(0, line), max(0, m.editor.LineCount()-1))
	for m.editor.Line() < line {
		m.editor.CursorDown()
	}
	for m.editor.Line() > line {
		m.editor.CursorUp()
	}
	m.editor.SetCursor(0)
}

func (m *model) editorPageUp() {
	target := max(0, m.editor.Line()-max(1, m.editor.Height()))
	m.moveEditorToLine(target)
}

func (m *model) editorPageDown() {
	target := min(max(0, m.editor.LineCount()-1), m.editor.Line()+max(1, m.editor.Height()))
	m.moveEditorToLine(target)
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
	if isConvertibleDocument(abs) {
		return m.importAndOpenDocument(abs)
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
	m.eyesOnly = false
	m.vaultID = ""
	m.expandTreeTo("file:" + abs)
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

func (m *model) importAndOpenDocument(path string) error {
	target := importPathForFile(path, m.cwd)
	if err := m.importFileToVault(path, target); err != nil {
		return err
	}
	if err := m.renderTree(); err != nil {
		return err
	}
	if err := m.openVaultPath(target); err != nil {
		return err
	}
	m.status = "imported and opened vault:" + target
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
	m.eyesOnly = note.EyesOnly
	if m.eyesOnly {
		m.mouseCapture = true
	}
	m.expandTreeTo("vault:" + note.Path)
	m.editor.SetValue(content)
	m.dirty = false
	m.status = "editing vault:" + note.Path
	if m.eyesOnly {
		m.status = "eyes only vault:" + note.Path
	}
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
	m.eyesOnly = m.eyesOnlyPaths[cleanVaultPath(path)]
	m.expandTreeTo("vault:" + path)
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
	m.expandTreeTo("file:" + abs)
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
	m.ensureTreeExpanded()
	selectedID := ""
	if len(m.tree) > 0 && m.treeIdx < len(m.tree) {
		selectedID = m.tree[m.treeIdx].id
	}
	if selectedID == "" {
		selectedID = m.currentTreeID()
	}
	notes, err := m.store.ListNotes()
	if err != nil {
		return err
	}
	vaultNotes := make([]string, 0, len(notes))
	m.eyesOnlyPaths = map[string]bool{}
	for _, note := range notes {
		vaultNotes = append(vaultNotes, note.Path)
		if note.EyesOnly {
			m.eyesOnlyPaths[cleanVaultPath(note.Path)] = true
		}
	}
	folders, err := m.store.ListFolders()
	if err != nil {
		return err
	}
	vaultFolders := make([]string, 0, len(folders))
	for _, folder := range folders {
		vaultFolders = append(vaultFolders, folder.Path)
	}
	tree, err := readTree(m.cwd, vaultNotes, vaultFolders, m.treeExpanded)
	m.tree = tree
	m.selectTreeID(selectedID)
	return err
}

func (m *model) ensureTreeExpanded() {
	if m.treeExpanded == nil {
		m.treeExpanded = map[string]bool{}
	}
	if _, ok := m.treeExpanded["vault:"]; !ok {
		m.treeExpanded["vault:"] = true
	}
	if _, ok := m.treeExpanded["file:"]; !ok {
		m.treeExpanded["file:"] = true
	}
}

func (m *model) selectTreeID(id string) {
	if id != "" {
		for i, entry := range m.tree {
			if entry.id == id {
				m.treeIdx = i
				m.ensureTreeSelectionVisible()
				return
			}
		}
	}
	if current := m.currentTreeID(); current != "" {
		for i, entry := range m.tree {
			if entry.id == current {
				m.treeIdx = i
				m.ensureTreeSelectionVisible()
				return
			}
		}
	}
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = max(0, len(m.tree)-1)
	}
	m.ensureTreeSelectionVisible()
}

func (m *model) ensureTreeSelectionVisible() {
	if len(m.tree) == 0 {
		m.treeIdx = 0
		m.treeOffset = 0
		return
	}
	if m.treeIdx < 0 {
		m.treeIdx = 0
	}
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = len(m.tree) - 1
	}
	height := m.treeContentHeight()
	if height <= 0 {
		height = 1
	}
	if m.treeIdx < m.treeOffset {
		m.treeOffset = m.treeIdx
	}
	if m.treeIdx >= m.treeOffset+height {
		m.treeOffset = m.treeIdx - height + 1
	}
	maxOffset := max(0, len(m.tree)-height)
	if m.treeOffset > maxOffset {
		m.treeOffset = maxOffset
	}
	if m.treeOffset < 0 {
		m.treeOffset = 0
	}
}

func (m model) treeContentHeight() int {
	treeW, _ := m.layoutWidths()
	if treeW <= 0 {
		return 0
	}
	return contentHeight(m.styles.sidebar, m.bodyHeight())
}

func (m model) currentTreeID() string {
	if m.isVault && m.vaultPath != "" {
		return "vault:" + cleanVaultPath(m.vaultPath)
	}
	if !m.isVault && m.diskPath != "" {
		return "file:" + m.diskPath
	}
	return ""
}

func (m *model) expandTreeTo(id string) {
	m.ensureTreeExpanded()
	switch {
	case id == "vault:" || strings.HasPrefix(id, "vault:"):
		m.treeExpanded["vault:"] = true
		path := strings.TrimPrefix(id, "vault:")
		parts := strings.Split(cleanVaultPath(path), "/")
		var prefix []string
		for i := 0; i < len(parts)-1; i++ {
			prefix = append(prefix, parts[i])
			m.treeExpanded["vault:"+strings.Join(prefix, "/")] = true
		}
	case id == "file:" || strings.HasPrefix(id, "file:"):
		m.treeExpanded["file:"] = true
		path := strings.TrimPrefix(id, "file:")
		dir := filepath.Dir(path)
		for dir != "." && dir != string(filepath.Separator) && strings.HasPrefix(dir, m.cwd) {
			m.treeExpanded["file:"+dir] = true
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
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
	m.helpView.Width = contentWidth(m.styles.panel, m.width)
	m.helpView.Height = contentHeight(m.styles.panel, innerH)
	m.markdown.Resize(m.preview.Width)
	if m.view == viewRender {
		m.renderPreview()
	}
}

package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlwrite/internal/config"
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

type selectPoint struct {
	row int
}

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
	selectionMode bool
	selecting     bool
	selectOffset  int
	selectStart   selectPoint
	selectEnd     selectPoint
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

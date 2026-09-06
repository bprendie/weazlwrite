package tui

import (
	"github.com/bprendie/weazlwrite/internal/config"
	textarea "github.com/bprendie/weazlwrite/internal/editorbuffer"
	"github.com/bprendie/weazlwrite/internal/storage"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"time"
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
	modeNewDocument
	modeConfirmDelete
	modeConfirmEyesOff
	modeRenameTree
	modeHelp
	modeFind
	modeJumpPage
	modeImporting
	modeLLMProvider
	modeLLMServer
	modeLLMLoading
	modeLLMModel
	modeLLMContext
	modeConfirmUnsaved
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

type textPos struct {
	line int
	col  int
}

func (a textPos) eq(b textPos) bool {
	return a.line == b.line && a.col == b.col
}

func (a textPos) less(b textPos) bool {
	return a.line < b.line || (a.line == b.line && a.col < b.col)
}

type model struct {
	search         editorSearchState
	diskTransition *pendingDiskTransition
	saves          vaultSaveState
	cfg            config.Config
	cfgPath        string
	store          *storage.Store
	styles         styles
	mode           mode
	focus          focus
	view           viewMode
	treeVisible    bool
	mouseCapture   bool
	width          int
	height         int
	password       textinput.Model
	confirmPass    textinput.Model
	vaultName      textinput.Model
	aiPrompt       textinput.Model
	filePrompt     textinput.Model
	vaultPrompt    textinput.Model
	folderPrompt   textinput.Model
	renamePrompt   textinput.Model
	findPrompt     textinput.Model
	jumpPrompt     textinput.Model
	llmPrompt      textinput.Model
	working        spinner.Model
	editor         textarea.Model
	editorChrome   *editorChrome
	preview        viewport.Model
	helpView       viewport.Model
	markdown       markdownRenderer
	tree           []treeEntry
	treeIdx        int
	treeOffset     int
	treeExpanded   map[string]bool
	eyesOnlyPaths  map[string]bool
	vaults         []vaultChoice
	vaultIdx       int
	activeVault    vaultChoice
	deleteTarget   treeEntry
	eyesOffTarget  treeEntry
	renameTarget   treeEntry
	newDocTarget   treeEntry
	carryTarget    treeEntry
	cwd            string
	filePath       string
	diskPath       string
	vaultPath      string
	vaultID        string
	autoNamed      bool
	isVault        bool
	eyesOnly       bool
	selectionMode  bool
	selecting      bool
	selectOffset   int
	selectStart    selectPoint
	selectEnd      selectPoint
	editorDrag     bool
	dragStart      textPos
	dragEnd        textPos
	undo           []editorSnapshot
	redo           []editorSnapshot
	undoOpen       bool
	lastEdit       time.Time
	lastEditLine   int
	lastEditKind   string
	editorEpoch    uint64
	dirty          bool
	aiBusy         bool
	generatingAt   time.Time
	lastFind       string
	pendingPass    string
	llmDraft       llmConfigDraft
	err            string
	status         string
}

type llmConfigDraft struct {
	ProviderType  string
	ServerURL     string
	Model         string
	ContextWindow int
	ProviderIndex int
	ModelIndex    int
	ContextIndex  int
	Models        []string
	FetchErr      string
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

type autoLockTickMsg struct{}

type llmModelsMsg struct {
	models []string
	err    error
}

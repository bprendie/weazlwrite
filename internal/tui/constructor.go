package tui

import (
	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

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

	llmPrompt := textinput.New()
	llmPrompt.Placeholder = "http://localhost:8000"
	llmPrompt.CharLimit = 4096

	s := newStyles()
	working := spinner.New(
		spinner.WithSpinner(spinner.Jump),
		spinner.WithStyle(s.status),
	)

	chrome := &editorChrome{gutter: editorGutterWidth}
	ta := newDocumentEditor()
	ta.SetPromptFunc(editorGutterWidth, chrome.prompt)
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
		llmPrompt:    llmPrompt,
		working:      working,
		editor:       ta,
		editorChrome: chrome,
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

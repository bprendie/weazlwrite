package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"time"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.EnableMouseCellMotion, autoLockTick())
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		m.renderPreview()
		m.renderHelp()
	case tea.MouseMsg:
		if m.enforceAutoLock() {
			return m, textinput.Blink
		}
		m.recordActivity()
		if m.mode == modeWrite && m.mouseCapture {
			return m.updateMouse(msg)
		}
	case tea.KeyMsg:
		if (msg.String() == "ctrl+c" && !m.editorTyping()) || msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
		if m.enforceAutoLock() {
			return m, textinput.Blink
		}
		m.recordActivity()
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
		if m.mode == modeNewDocument {
			return m.updateNewDocument(msg)
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
		if m.isLLMConfigMode() {
			return m.updateLLMConfig(msg)
		}
		if m.mode == modeImporting {
			return m, nil
		}
		if m.mode == modeGenerating {
			return m, nil
		}
		return m.updateWrite(msg)
	case editorCopyMsg:
		return m.finishEditorCopy(msg)
	case editorPasteMsg:
		if msg.epoch != m.editorEpoch || m.mode != modeWrite || !m.editorTyping() {
			return m, nil
		}
		if msg.err != nil {
			m.err = "paste: " + msg.err.Error()
			return m, nil
		}
		return m.updateEditorInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(msg.text), Paste: true})
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
		m.undoOpen = false
		m.recordEdit(m.editorSnapshot())
		m.editor.InsertString("\n\n" + block + "\n\n")
		m.undoOpen = false
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
	case llmModelsMsg:
		return m.handleLLMModelsMsg(msg)
	case spinner.TickMsg:
		if m.mode == modeGenerating || m.mode == modeImporting || m.mode == modeLLMLoading || m.aiBusy {
			var cmd tea.Cmd
			m.working, cmd = m.working.Update(msg)
			return m, cmd
		}
	case autoLockTickMsg:
		if m.enforceAutoLock() {
			return m, tea.Batch(textinput.Blink, autoLockTick())
		}
		return m, autoLockTick()
	default:
		// Cursor blink and asynchronous widget messages must reach their owner.
		var cmd tea.Cmd
		switch m.mode {
		case modeWrite:
			if m.editorTyping() {
				m.editor, cmd = m.editor.Update(msg)
			}
		case modeVault:
			m.password, cmd = m.password.Update(msg)
		case modeVaultConfirm:
			m.confirmPass, cmd = m.confirmPass.Update(msg)
		case modeVaultName:
			m.vaultName, cmd = m.vaultName.Update(msg)
		case modeFind:
			if m.view == viewEdit && m.search.replace {
				m.search.replacement, cmd = m.search.replacement.Update(msg)
			} else {
				before := m.findPrompt.Value()
				m.findPrompt, cmd = m.findPrompt.Update(msg)
				if m.view == viewEdit && before != m.findPrompt.Value() {
					m.refreshEditorMatches(m.search.origin)
				}
			}
		case modeJumpPage:
			m.jumpPrompt, cmd = m.jumpPrompt.Update(msg)
		case modeAI:
			m.aiPrompt, cmd = m.aiPrompt.Update(msg)
		case modeSaveFile:
			m.filePrompt, cmd = m.filePrompt.Update(msg)
		case modeSaveVault:
			m.vaultPrompt, cmd = m.vaultPrompt.Update(msg)
		case modeNewFolder:
			m.folderPrompt, cmd = m.folderPrompt.Update(msg)
		case modeNewDocument, modeRenameTree:
			m.renamePrompt, cmd = m.renamePrompt.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

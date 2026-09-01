package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m model) statusView(width int) string {
	if m.err != "" {
		text := "! " + strings.ReplaceAll(m.err, "\n", " ")
		return m.styles.error.Inline(true).MaxWidth(width).Render(minString(text, max(1, width)))
	}
	mode := strings.ToUpper(m.viewName())
	if m.mode != modeWrite {
		mode = strings.ToUpper(m.modeName())
	}
	parts := []string{m.styles.statusMode.Render(mode)}
	if m.eyesOnly {
		parts = append(parts, m.styles.statusDirty.Render("EYES"))
	}
	if m.dirty {
		parts = append(parts, m.styles.statusDirty.Render("*"))
	}
	target := m.targetLabel()
	if target != "" {
		parts = append(parts, m.styles.statusPath.Render(target))
	}
	status := strings.ReplaceAll(m.status, "\n", " ")
	if m.mode == modeWrite {
		status = strings.TrimSpace(status + fmt.Sprintf("  page %d/%d", m.currentPage(), m.totalPages()))
	}
	if status != "" {
		parts = append(parts, m.styles.status.Render(status))
	}
	line := strings.Join(parts, " ")
	return m.styles.statusBar.Inline(true).MaxWidth(width).Render(ansi.Truncate(line, max(1, width), ""))
}

func (m model) helpFooterView(width int) string {
	return m.styles.help.Inline(true).MaxWidth(width).Render(ansi.Truncate(m.highlightHelpKeys(m.helpText()), max(1, width), ""))
}

func (m model) highlightHelpKeys(text string) string {
	parts := strings.Split(text, " | ")
	for i, part := range parts {
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		key := fields[0]
		rest := strings.TrimSpace(strings.TrimPrefix(part, key))
		if rest == "" {
			parts[i] = m.styles.helpKey.Render(key)
			continue
		}
		parts[i] = m.styles.helpKey.Render(key) + " " + rest
	}
	return strings.Join(parts, m.styles.help.Render(" | "))
}

func (m model) viewName() string {
	if m.view == viewRender {
		return "render"
	}
	return "edit"
}

func (m model) modeName() string {
	switch m.mode {
	case modeVaultPicker:
		return "vaults"
	case modeVaultName:
		return "new vault"
	case modeVault:
		return "unlock"
	case modeVaultConfirm:
		return "confirm"
	case modeAI:
		return "ai"
	case modeGenerating:
		return "ai"
	case modeSaveFile:
		return "save file"
	case modeSaveVault:
		return "save vault"
	case modeNewFolder:
		return "folder"
	case modeConfirmDelete:
		return "delete"
	case modeConfirmEyesOff:
		return "eyes"
	case modeRenameTree:
		return "rename"
	case modeHelp:
		return "commands"
	case modeFind:
		return "find"
	case modeJumpPage:
		return "jump"
	case modeImporting:
		return "import"
	case modeLLMProvider:
		return "llm"
	case modeLLMServer:
		return "llm"
	case modeLLMLoading:
		return "llm"
	case modeLLMModel:
		return "llm"
	case modeLLMContext:
		return "llm"
	default:
		return m.viewName()
	}
}

func (m model) targetLabel() string {
	if m.isVault && m.vaultPath != "" {
		return "vault:" + m.vaultPath
	}
	if !m.isVault && m.filePath != "" {
		return "disk:" + m.filePath
	}
	return ""
}

func (m model) helpText() string {
	if m.mode == modeVaultPicker {
		return "up/down select | enter open | n new vault | ctrl+c quit"
	}
	if m.mode == modeVaultName {
		return "enter create | esc cancel | ctrl+c quit"
	}
	if m.mode == modeVault {
		return "enter unlock/create | ctrl+c quit"
	}
	if m.mode == modeVaultConfirm {
		return "enter create | esc restart password | ctrl+c quit"
	}
	if m.mode == modeAI {
		return "enter generate | esc cancel | ctrl+c quit"
	}
	if m.mode == modeGenerating {
		return "waiting for local model | ctrl+c quit"
	}
	if m.mode == modeWrite && m.selectionMode {
		return "selection mode | drag in writing pane to copy | esc/^Y cancel | ctrl+c quit"
	}
	if m.mode == modeSaveFile {
		return "enter save | esc cancel | ctrl+c quit"
	}
	if m.mode == modeSaveVault {
		return "enter save encrypted | esc cancel | ctrl+c quit"
	}
	if m.mode == modeNewFolder {
		return "enter create | esc cancel | ctrl+c quit"
	}
	if m.mode == modeNewDocument {
		return "enter create | esc cancel | ctrl+c quit"
	}
	if m.mode == modeConfirmDelete {
		return "enter/y delete | esc/n cancel | ctrl+c quit"
	}
	if m.mode == modeConfirmEyesOff {
		return "enter/y disable eyes only | esc/n cancel | ctrl+c quit"
	}
	if m.mode == modeRenameTree {
		return "enter rename | esc cancel | ctrl+c quit"
	}
	if m.mode == modeHelp {
		return "up/down scroll | pgup/pgdown | esc close | ctrl+c quit"
	}
	if m.mode == modeFind {
		return "enter find next | esc cancel | ctrl+c quit"
	}
	if m.mode == modeJumpPage {
		return "enter jump | esc cancel | ctrl+c quit"
	}
	if m.mode == modeImporting {
		return "importing in background | ctrl+c quit"
	}
	if m.isLLMConfigMode() {
		if m.mode == modeLLMServer || (m.mode == modeLLMModel && (m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0)) {
			return "enter continue | esc cancel | ctrl+c quit"
		}
		if m.mode == modeLLMLoading {
			return "fetching models | esc cancel | ctrl+c quit"
		}
		return "up/down select | enter continue | esc cancel | ctrl+c quit"
	}
	target := "vault"
	if !m.isVault && m.filePath != "" {
		target = fmt.Sprintf("disk:%s", m.filePath)
	}
	mode := "edit"
	if m.view == viewRender {
		mode = "render"
	}
	tree := "tree:on"
	if !m.treeVisible {
		tree = "tree:off"
	}
	mouse := "mouse:on"
	if !m.mouseCapture {
		mouse = "mouse:off"
	}
	eyes := ""
	if m.eyesOnly {
		eyes = " eyes-only"
	}
	return mode + eyes + " " + tree + " " + mouse + " | " + keyCycleFocus + " focus | enter open | ^S " + target + " | " + keyAI + " AI | " + keyLLM + " llm | " + keyEyes + " eyes | " + keyHelp + " commands | ctrl+c"
}

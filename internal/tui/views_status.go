package tui

import "fmt"

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
	if m.selectionMode {
		mouse = "select:on"
	}
	eyes := ""
	if m.eyesOnly {
		eyes = " eyes-only"
	}
	return mode + eyes + " " + tree + " " + mouse + " | tab focus | enter open | ^S " + target + " | ^P AI | alt+O eyes | ^K commands | ^C"
}

package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

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

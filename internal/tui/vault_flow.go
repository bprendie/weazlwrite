package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/storage"
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

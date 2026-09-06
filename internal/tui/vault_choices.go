package tui

import (
	"errors"
	"fmt"
	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/storage"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
	store.SetAutoLockTimeout(m.cfg.Vault.AutoLockTimeout())
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

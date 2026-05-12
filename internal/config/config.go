package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const appName = "weazlwrite"

type Config struct {
	ActiveProvider string              `json:"active_provider"`
	Providers      map[string]Provider `json:"providers"`
	Database       Database            `json:"database"`
	Vault          Vault               `json:"vault"`
	UI             UI                  `json:"ui"`
}

type Provider struct {
	Type          string `json:"type"`
	ServerURL     string `json:"server_url"`
	Model         string `json:"model"`
	APIKey        string `json:"api_key,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`
}

type Database struct {
	Path string `json:"path"`
}

type Vault struct {
	Root string `json:"root"`
}

type UI struct {
	RenderMarkdown *bool  `json:"render_markdown,omitempty"`
	MarkdownStyle  string `json:"markdown_style,omitempty"`
}

func Load() (Config, string, error) {
	path := configPath()
	cfg, err := LoadPath(path)
	return cfg, path, err
}

func LoadPath(path string) (Config, error) {
	cfg := Default()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o700); err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(cfg.Vault.Root, 0o700); err != nil {
		return cfg, err
	}

	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, Save(path, cfg)
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.withDefaults()
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o700); err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(cfg.Vault.Root, 0o700); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.withDefaults()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func Default() Config {
	dataDir := dataDir()
	return Config{
		ActiveProvider: "local-vllm",
		Providers: map[string]Provider{
			"local-vllm": {
				Type:          "vllm",
				ServerURL:     "http://localhost:8000",
				Model:         "local-model",
				ContextWindow: 32768,
			},
			"local-ollama": {
				Type:          "ollama",
				ServerURL:     "http://localhost:11434",
				Model:         "llama3.1",
				ContextWindow: 32768,
			},
		},
		Database: Database{Path: filepath.Join(dataDir, "vault", "weazlwrite.sqlite3")},
		Vault:    Vault{Root: filepath.Join(dataDir, "vault")},
		UI: UI{
			RenderMarkdown: boolPtr(true),
			MarkdownStyle:  "dark",
		},
	}
}

func (c *Config) withDefaults() {
	def := Default()
	if c.ActiveProvider == "" {
		c.ActiveProvider = def.ActiveProvider
	}
	if c.Providers == nil || len(c.Providers) == 0 {
		c.Providers = def.Providers
	}
	for name, provider := range c.Providers {
		if provider.ContextWindow <= 0 {
			provider.ContextWindow = 32768
			c.Providers[name] = provider
		}
	}
	if c.Database.Path == "" {
		c.Database.Path = def.Database.Path
	}
	if c.Vault.Root == "" {
		c.Vault.Root = def.Vault.Root
	}
	legacy := legacyDefault()
	if c.Database.Path == legacy.Database.Path {
		c.Database.Path = def.Database.Path
	}
	if c.Vault.Root == legacy.Vault.Root {
		c.Vault.Root = def.Vault.Root
	}
	if c.UI.RenderMarkdown == nil {
		c.UI.RenderMarkdown = def.UI.RenderMarkdown
	}
	if c.UI.MarkdownStyle == "" {
		c.UI.MarkdownStyle = def.UI.MarkdownStyle
	}
}

func (c *Config) Active() Provider {
	if c.Providers == nil {
		c.Providers = map[string]Provider{}
	}
	p, ok := c.Providers[c.ActiveProvider]
	if !ok {
		return Provider{}
	}
	return p
}

func (ui UI) MarkdownEnabled() bool {
	return ui.RenderMarkdown == nil || *ui.RenderMarkdown
}

func boolPtr(v bool) *bool {
	return &v
}

func configPath() string {
	if p := os.Getenv("WEAZLWRITE_CONFIG"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName, "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName, "config.json")
}

func dataDir() string {
	if p := os.Getenv("WEAZLWRITE_DATA"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+appName)
}

func legacyDefault() Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".local", "share", appName)
	return Config{
		Database: Database{Path: filepath.Join(dataDir, "weazlwrite.sqlite3")},
		Vault:    Vault{Root: filepath.Join(dataDir, "vault")},
	}
}

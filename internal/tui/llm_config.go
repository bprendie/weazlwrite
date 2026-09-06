package tui

import (
	"context"
	"fmt"
	"github.com/bprendie/weazlwrite/internal/config"
	"github.com/bprendie/weazlwrite/internal/llm"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strconv"
	"strings"
	"time"
)

var contextWindowChoices = []struct {
	name   string
	tokens int
	note   string
}{
	{name: "small", tokens: 8192},
	{name: "medium", tokens: 16384},
	{name: "large", tokens: 32768},
	{name: "xl", tokens: 128000, note: "large local servers only"},
}

func (m model) isLLMConfigMode() bool {
	return m.mode == modeLLMProvider ||
		m.mode == modeLLMServer ||
		m.mode == modeLLMLoading ||
		m.mode == modeLLMModel ||
		m.mode == modeLLMContext
}

func (m model) startLLMConfig() (tea.Model, tea.Cmd) {
	p := m.cfg.Active()
	providerType := p.Type
	if providerType != "ollama" {
		providerType = "vllm"
	}
	contextWindow := p.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	m.llmDraft = llmConfigDraft{
		ProviderType:  providerType,
		ServerURL:     p.ServerURL,
		Model:         p.Model,
		ContextWindow: contextWindow,
		ProviderIndex: providerIndex(providerType),
		ContextIndex:  contextChoiceIndex(contextWindow),
	}
	m.llmPrompt.Reset()
	m.llmPrompt.Blur()
	m.editor.Blur()
	m.mode = modeLLMProvider
	m.status = "select llm provider"
	m.err = ""
	return m, nil
}

func (m model) updateLLMConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.cancelLLMConfig()
	case "ctrl+c":
		return m, tea.Quit
	}
	switch m.mode {
	case modeLLMProvider:
		return m.updateLLMProvider(msg)
	case modeLLMServer:
		return m.updateLLMServer(msg)
	case modeLLMLoading:
		return m, nil
	case modeLLMModel:
		return m.updateLLMModel(msg)
	case modeLLMContext:
		return m.updateLLMContext(msg)
	}
	return m, nil
}

func (m model) updateLLMProvider(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ProviderIndex = max(0, m.llmDraft.ProviderIndex-1)
	case "down", "right":
		m.llmDraft.ProviderIndex = min(1, m.llmDraft.ProviderIndex+1)
	case "1":
		m.llmDraft.ProviderIndex = 0
	case "2":
		m.llmDraft.ProviderIndex = 1
	case "enter":
		m.llmDraft.ProviderType = []string{"vllm", "ollama"}[m.llmDraft.ProviderIndex]
		if m.llmDraft.ServerURL == "" || m.cfg.Active().Type != m.llmDraft.ProviderType {
			m.llmDraft.ServerURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
		}
		m.llmPrompt.SetValue(llm.NormalizeServerURL(m.llmDraft.ProviderType, m.llmDraft.ServerURL))
		m.llmPrompt.CursorEnd()
		m.llmPrompt.Placeholder = llm.DefaultServerURL(m.llmDraft.ProviderType)
		m.llmPrompt.Focus()
		m.mode = modeLLMServer
		m.status = "set llm server"
		return m, textinput.Blink
	}
	return m, nil
}

func (m model) updateLLMServer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "enter" {
		var cmd tea.Cmd
		m.llmPrompt, cmd = m.llmPrompt.Update(msg)
		return m, cmd
	}
	serverURL := llm.NormalizeServerURL(m.llmDraft.ProviderType, m.llmPrompt.Value())
	if serverURL == "" {
		serverURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
	}
	m.llmDraft.ServerURL = serverURL
	m.llmDraft.Models = nil
	m.llmDraft.FetchErr = ""
	m.llmDraft.ModelIndex = 0
	m.llmPrompt.Reset()
	m.llmPrompt.Blur()
	m.mode = modeLLMLoading
	m.status = "querying models"
	return m, tea.Batch(m.fetchLLMModels(), m.working.Tick)
}

func (m model) fetchLLMModels() tea.Cmd {
	providerType := m.llmDraft.ProviderType
	serverURL := m.llmDraft.ServerURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := llm.FetchModels(ctx, providerType, serverURL)
		return llmModelsMsg{models: models, err: err}
	}
}

func (m model) handleLLMModelsMsg(msg llmModelsMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeLLMLoading {
		return m, nil
	}
	m.llmDraft.Models = msg.models
	if msg.err != nil {
		m.llmDraft.FetchErr = msg.err.Error()
		m.llmPrompt.SetValue(defaultDraftModel(m.llmDraft.ProviderType, m.cfg.Active().Model))
		m.llmPrompt.CursorEnd()
		m.llmPrompt.Placeholder = "model name"
		m.llmPrompt.Focus()
		m.status = "enter model manually"
		m.mode = modeLLMModel
		return m, textinput.Blink
	}
	if len(msg.models) == 0 {
		m.llmDraft.FetchErr = "provider returned no models"
		m.llmPrompt.SetValue(defaultDraftModel(m.llmDraft.ProviderType, m.cfg.Active().Model))
		m.llmPrompt.CursorEnd()
		m.llmPrompt.Placeholder = "model name"
		m.llmPrompt.Focus()
		m.status = "enter model manually"
		m.mode = modeLLMModel
		return m, textinput.Blink
	}
	m.llmDraft.FetchErr = ""
	m.llmDraft.ModelIndex = modelChoiceIndex(msg.models, m.cfg.Active().Model)
	m.status = "select model"
	m.mode = modeLLMModel
	return m, nil
}

func (m model) updateLLMModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		if msg.String() == "enter" {
			modelName := strings.TrimSpace(m.llmPrompt.Value())
			if modelName == "" {
				m.err = "model name is required"
				return m, nil
			}
			m.llmDraft.Model = modelName
			m.llmPrompt.Reset()
			m.llmPrompt.Blur()
			m.mode = modeLLMContext
			m.status = "select context window"
			m.err = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.llmPrompt, cmd = m.llmPrompt.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ModelIndex = max(0, m.llmDraft.ModelIndex-1)
	case "down", "right":
		m.llmDraft.ModelIndex = min(len(m.llmDraft.Models)-1, m.llmDraft.ModelIndex+1)
	case "enter":
		m.llmDraft.Model = m.llmDraft.Models[m.llmDraft.ModelIndex]
		m.mode = modeLLMContext
		m.status = "select context window"
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(m.llmDraft.Models) {
			m.llmDraft.ModelIndex = n - 1
		}
	}
	return m, nil
}

func (m model) updateLLMContext(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ContextIndex = max(0, m.llmDraft.ContextIndex-1)
	case "down", "right":
		m.llmDraft.ContextIndex = min(len(contextWindowChoices)-1, m.llmDraft.ContextIndex+1)
	case "enter":
		m.llmDraft.ContextWindow = contextWindowChoices[m.llmDraft.ContextIndex].tokens
		return m.saveLLMConfig()
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(contextWindowChoices) {
			m.llmDraft.ContextIndex = n - 1
		}
	}
	return m, nil
}

func (m model) saveLLMConfig() (tea.Model, tea.Cmd) {
	providerType := m.llmDraft.ProviderType
	providerID := "primary-" + providerType
	if m.cfg.Providers == nil {
		m.cfg.Providers = map[string]config.Provider{}
	}
	m.cfg.ActiveProvider = providerID
	m.cfg.Providers[providerID] = config.Provider{
		Type:          providerType,
		ServerURL:     llm.NormalizeServerURL(providerType, m.llmDraft.ServerURL),
		Model:         m.llmDraft.Model,
		ContextWindow: m.llmDraft.ContextWindow,
	}
	if err := config.Save(m.cfgPath, m.cfg); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.llmPrompt.Reset()
	m.llmPrompt.Blur()
	m.llmDraft = llmConfigDraft{}
	m.mode = modeWrite
	m.setMainFocus()
	m.err = ""
	m.status = fmt.Sprintf("llm set: %s %s", providerType, m.cfg.Active().Model)
	return m, nil
}

func (m model) cancelLLMConfig() (tea.Model, tea.Cmd) {
	m.llmPrompt.Reset()
	m.llmPrompt.Blur()
	m.llmDraft = llmConfigDraft{}
	m.mode = modeWrite
	m.setMainFocus()
	m.err = ""
	m.status = "llm config canceled"
	return m, nil
}

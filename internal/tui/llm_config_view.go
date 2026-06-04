package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bprendie/weazlwrite/internal/config"
)

func (m model) llmConfigView() string {
	switch m.mode {
	case modeLLMProvider:
		return m.modalView("LLM provider", m.providerChoicesView(), "", neonCyan, 82)
	case modeLLMServer:
		body := strings.Join([]string{
			m.styles.help.Render(serverHint(m.llmDraft.ProviderType)),
			"",
			m.llmPrompt.View(),
		}, "\n")
		return m.modalView("LLM server", body, "", neonCyan, 86)
	case modeLLMLoading:
		body := fmt.Sprintf("%s fetching %s models from %s", m.working.View(), m.llmDraft.ProviderType, m.llmDraft.ServerURL)
		return m.modalView("LLM models", body, "", neonViolet, 86)
	case modeLLMModel:
		return m.modalView("LLM model", m.modelChoicesView(), "", neonCyan, 92)
	case modeLLMContext:
		return m.modalView("Context window", m.contextChoicesView(), currentLLMHelp(m.cfg.Active()), neonViolet, 86)
	default:
		return ""
	}
}

func (m model) providerChoicesView() string {
	choices := []string{"vllm", "ollama"}
	rows := make([]string, 0, len(choices))
	for i, choice := range choices {
		rows = append(rows, m.selectRow(i == m.llmDraft.ProviderIndex, fmt.Sprintf("%d", i+1), choice, providerDescription(choice)))
	}
	rows = append(rows, "", currentLLMHelp(m.cfg.Active()))
	return strings.Join(rows, "\n")
}

func (m model) modelChoicesView() string {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		return strings.Join([]string{
			m.styles.error.Render(m.llmDraft.FetchErr),
			"",
			m.styles.help.Render("Type the model name exactly as your local server expects."),
			"",
			m.llmPrompt.View(),
		}, "\n")
	}
	rows := make([]string, 0, len(m.llmDraft.Models))
	for i, modelName := range m.llmDraft.Models {
		rows = append(rows, m.selectRow(i == m.llmDraft.ModelIndex, fmt.Sprintf("%d", i+1), modelName, ""))
	}
	return strings.Join(rows, "\n")
}

func (m model) contextChoicesView() string {
	rows := make([]string, 0, len(contextWindowChoices))
	for i, choice := range contextWindowChoices {
		desc := fmt.Sprintf("%d tokens", choice.tokens)
		if choice.note != "" {
			desc += " - " + choice.note
		}
		rows = append(rows, m.selectRow(i == m.llmDraft.ContextIndex, fmt.Sprintf("%d", i+1), choice.name, desc))
	}
	return strings.Join(rows, "\n")
}

func (m model) selectRow(selected bool, key, label, desc string) string {
	marker := " "
	markerStyle := m.styles.help
	labelStyle := m.styles.modalBody
	if selected {
		marker = ">"
		markerStyle = m.styles.status
		labelStyle = m.styles.statusDirty
	}
	left := lipgloss.JoinHorizontal(
		lipgloss.Top,
		markerStyle.Render(marker),
		" ",
		m.styles.helpKey.Render(key),
		" ",
		labelStyle.Render(label),
	)
	if desc == "" {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", m.styles.help.Render(desc))
}

func providerDescription(providerType string) string {
	if providerType == "ollama" {
		return "local Ollama /api/chat"
	}
	return "OpenAI-compatible local /v1/chat/completions"
}

func serverHint(providerType string) string {
	if providerType == "ollama" {
		return "Base URL only, without /api. Example: http://localhost:11434"
	}
	return "Base URL only, without /v1. Example: http://localhost:8000"
}

func currentLLMHelp(provider config.Provider) string {
	contextWindow := provider.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	return fmt.Sprintf("Current: %s / %s / %d tokens", provider.Type, provider.Model, contextWindow)
}

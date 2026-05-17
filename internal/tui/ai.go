package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlwrite/internal/llm"
)

func (m model) startAIInsert() (tea.Model, tea.Cmd) {
	m.mode = modeAI
	m.aiPrompt.SetValue("")
	m.aiPrompt.Focus()
	m.editor.Blur()
	m.status = "ai intelligence prompt"
	return m, textinput.Blink
}

func (m model) updateAI(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		instruction := strings.TrimSpace(m.aiPrompt.Value())
		if instruction == "" {
			m.err = "ai prompt is required"
			return m, nil
		}
		m.mode = modeGenerating
		m.aiBusy = true
		m.generatingAt = time.Now()
		m.err = ""
		m.status = "ai generating block"
		return m, tea.Batch(m.generateAIBlock(instruction, m.editor.Value()), m.working.Tick)
	case "esc":
		m.mode = modeWrite
		m.setMainFocus()
		m.status = "ai insert cancelled"
		return m, nil
	default:
		var cmd tea.Cmd
		m.aiPrompt, cmd = m.aiPrompt.Update(msg)
		return m, cmd
	}
}

func (m model) generateAIBlock(instruction, document string) tea.Cmd {
	provider := m.cfg.Active()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		block, err := llm.New(provider).GenerateBlock(ctx, document, instruction)
		return aiResultMsg{block: block, err: err}
	}
}

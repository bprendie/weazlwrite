package tui

import (
	"github.com/bprendie/weazlwrite/internal/llm"
	"strings"
)

func providerIndex(providerType string) int {
	if providerType == "ollama" {
		return 1
	}
	return 0
}

func contextChoiceIndex(tokens int) int {
	best := 0
	for i, choice := range contextWindowChoices {
		if choice.tokens == tokens {
			return i
		}
		if choice.tokens <= tokens {
			best = i
		}
	}
	return best
}

func modelChoiceIndex(models []string, current string) int {
	for i, model := range models {
		if model == current {
			return i
		}
	}
	return 0
}

func defaultDraftModel(providerType, current string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return llm.DefaultModel(providerType)
}
